// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// code_review.go: Complete vertical slice for code review orchestration
// - Multi-reviewer voting (unanimous approval required)
// - Specialized reviewers (testing, functional, architecture)
// - Vote parsing and aggregation
//
// This slice follows CUPID principles:
// - Composable: Self-contained review orchestration
// - Unix philosophy: Does code review coordination, nothing else
// - Predictable: Clear approval/rejection with feedback
// - Idiomatic: Review conventions, Temporal patterns
// - Domain-centric: Organized around review capability
package slices

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
)

// ============================================================================
// ACTIVITIES
// ============================================================================

// CodeReviewActivities handles all code review operations
type CodeReviewActivities struct {
	// No external dependencies - uses SDK client from bootstrap output
}

// NewCodeReviewActivities creates a new code review activities instance
func NewCodeReviewActivities() *CodeReviewActivities {
	return &CodeReviewActivities{}
}

// ExecuteReview runs a single reviewer's code review
//
// This activity:
// 1. Reconstructs SDK client from bootstrap output
// 2. Executes review prompt via SDK with reviewer specialization
// 3. Parses vote (APPROVE/REQUEST_CHANGE/REJECT) from response
// 4. Returns ReviewVote with feedback
//
// Specialized reviewers:
// - Testing: Focuses on test coverage, quality, edge cases
// - Functional: Focuses on correctness, logic, requirements
// - Architecture: Focuses on design, patterns, maintainability
func (c *CodeReviewActivities) ExecuteReview(ctx context.Context, output BootstrapOutput, taskInput TaskInput, reviewType ReviewType) (*ReviewVote, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing code review", "cellID", output.CellID, "reviewType", reviewType)

	activity.RecordHeartbeat(ctx, fmt.Sprintf("reviewing as %s", reviewType))

	startTime := time.Now()

	// Reconstruct SDK client
	client := ReconstructClient(output)

	// Build review prompt based on specialization
	prompt := buildReviewPrompt(taskInput, reviewType)

	// Execute review via SDK
	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Agent: "review",
		Title: fmt.Sprintf("%s Review - %s", reviewType, taskInput.TaskID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute review in cell %q: %w", output.CellID, err)
	}

	duration := time.Since(startTime)

	// Parse vote from response
	parsedVote := parseReviewVote(result.GetText())

	reviewVote := &ReviewVote{
		ReviewerName: fmt.Sprintf("reviewer-%s", reviewType),
		ReviewType:   reviewType,
		Vote:         parsedVote.Vote,
		Feedback:     parsedVote.Feedback,
		Duration:     duration,
	}

	logger.Info("Review completed",
		"cellID", output.CellID,
		"reviewType", reviewType,
		"vote", reviewVote.Vote,
		"duration", duration)

	return reviewVote, nil
}

// ExecuteMultiReview runs multiple reviewers in parallel and aggregates votes
//
// This activity implements Gate 6 of Enhanced TCR:
// 1. Execute 3 reviewers with different specializations
// 2. Require unanimous APPROVE vote
// 3. Any REQUEST_CHANGE or REJECT fails the gate
//
// Returns GateResult with all review votes and final decision.
func (c *CodeReviewActivities) ExecuteMultiReview(ctx context.Context, output BootstrapOutput, taskInput TaskInput, reviewersCount int) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing multi-reviewer approval", "cellID", output.CellID, "reviewersCount", reviewersCount)

	startTime := time.Now()

	// Default to 3 reviewers if not specified
	if reviewersCount == 0 {
		reviewersCount = 3
	}

	// Define review types to execute
	reviewTypes := []ReviewType{
		ReviewTypeTesting,
		ReviewTypeFunctional,
		ReviewTypeArchitecture,
	}

	// Limit to requested count
	if reviewersCount < len(reviewTypes) {
		reviewTypes = reviewTypes[:reviewersCount]
	}

	// Execute reviews (sequentially for now - could be parallel in production)
	var votes []ReviewVote
	var errors []string

	for _, reviewType := range reviewTypes {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("executing %s review", reviewType))

		vote, err := c.ExecuteReview(ctx, output, taskInput, reviewType)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s review failed: %v", reviewType, err))
			// Create placeholder vote for failed review
			votes = append(votes, ReviewVote{
				ReviewerName: fmt.Sprintf("reviewer-%s", reviewType),
				ReviewType:   reviewType,
				Vote:         VoteReject,
				Feedback:     fmt.Sprintf("Review failed: %v", err),
			})
			continue
		}

		votes = append(votes, *vote)
	}

	duration := time.Since(startTime)

	// Aggregate votes - require unanimous approval
	passed, message := aggregateVotes(votes)

	// Check for execution errors
	if len(errors) > 0 {
		passed = false
		message = fmt.Sprintf("Review execution errors: %v; %s", errors, message)
	}

	logger.Info("Multi-review completed",
		"cellID", output.CellID,
		"passed", passed,
		"votes", len(votes),
		"duration", duration)

	return &GateResult{
		GateName:    "multi_review",
		Passed:      passed,
		ReviewVotes: votes,
		Duration:    duration,
		Message:     message,
	}, nil
}

// RequestChanges sends feedback to the agent requesting code changes
//
// This activity is called when reviews fail:
// 1. Aggregates all reviewer feedback
// 2. Sends consolidated feedback to agent
// 3. Agent makes changes
// 4. Returns updated code for re-review
func (c *CodeReviewActivities) RequestChanges(ctx context.Context, output BootstrapOutput, votes []ReviewVote) (*TaskOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Requesting code changes", "cellID", output.CellID)

	activity.RecordHeartbeat(ctx, "processing reviewer feedback")

	// Reconstruct SDK client
	client := ReconstructClient(output)

	// Build consolidated feedback
	feedback := consolidateFeedback(votes)

	// Send feedback to agent
	prompt := fmt.Sprintf(`The code review process has requested changes. Please address the following feedback:

%s

Make the necessary changes to address all reviewer concerns.`, feedback)

	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Agent: "build",
		Title: "Address Review Feedback",
	})
	if err != nil {
		return &TaskOutput{
			Success: false,
			Error:   err.Error(),
		}, fmt.Errorf("failed to request changes in cell %q: %w", output.CellID, err)
	}

	// Extract modified files
	filesModified := extractModifiedFiles(result)

	logger.Info("Changes requested and applied",
		"cellID", output.CellID,
		"filesModified", len(filesModified))

	return &TaskOutput{
		Success:       true,
		Output:        result.GetText(),
		FilesModified: filesModified,
	}, nil
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// buildReviewPrompt creates a review prompt based on specialization
func buildReviewPrompt(taskInput TaskInput, reviewType ReviewType) string {
	var focus string
	var criteria string

	switch reviewType {
	case ReviewTypeTesting:
		focus = "test coverage and quality"
		criteria = `- Are all edge cases covered?
- Are tests comprehensive and meaningful?
- Do tests follow TDD principles?
- Is test code maintainable?
- Are assertions clear and specific?`

	case ReviewTypeFunctional:
		focus = "functional correctness and requirements"
		criteria = `- Does the implementation meet requirements?
- Is the logic correct and complete?
- Are errors handled appropriately?
- Are there any bugs or logic errors?
- Does it follow Go best practices?`

	case ReviewTypeArchitecture:
		focus = "architecture and design"
		criteria = `- Does it follow SOLID principles?
- Is the code maintainable and extensible?
- Are abstractions appropriate?
- Does it integrate well with existing code?
- Are there any design smells?`

	default:
		focus = "code quality"
		criteria = "- General code quality assessment"
	}

	prompt := fmt.Sprintf(`You are a code reviewer specializing in %s.

Task: %s
Description: %s

Review the changes and provide your assessment:

Evaluation Criteria:
%s

Your response MUST end with one of:
- "VOTE: APPROVE" - if all criteria are met
- "VOTE: REQUEST_CHANGE" - if changes are needed
- "VOTE: REJECT" - if fundamental issues exist

Provide specific feedback before your vote.

Prompt: %s`, focus, taskInput.TaskID, taskInput.Description, criteria, taskInput.Prompt)

	return prompt
}

// parseReviewVote extracts vote and feedback from review response
func parseReviewVote(response string) ParsedVote {
	// Default to reject if parsing fails
	vote := VoteReject
	feedback := response

	// Look for vote markers in response
	if containsString(response, "VOTE: APPROVE") || containsString(response, "APPROVE") {
		vote = VoteApprove
	} else if containsString(response, "VOTE: REQUEST_CHANGE") || containsString(response, "REQUEST_CHANGE") {
		vote = VoteRequestChange
	} else if containsString(response, "VOTE: REJECT") || containsString(response, "REJECT") {
		vote = VoteReject
	}

	// Extract feedback (everything before VOTE: marker)
	voteIdx := indexString(response, "VOTE:")
	if voteIdx > 0 {
		feedback = response[:voteIdx]
		feedback = trimWhitespace(feedback)
	}

	return ParsedVote{
		Vote:     vote,
		Feedback: feedback,
	}
}

// aggregateVotes determines if reviews pass based on unanimous approval
func aggregateVotes(votes []ReviewVote) (bool, string) {
	if len(votes) == 0 {
		return false, "No reviews received"
	}

	approveCount := 0
	requestChangeCount := 0
	rejectCount := 0

	for _, vote := range votes {
		switch vote.Vote {
		case VoteApprove:
			approveCount++
		case VoteRequestChange:
			requestChangeCount++
		case VoteReject:
			rejectCount++
		}
	}

	// Require unanimous approval
	if approveCount == len(votes) {
		return true, fmt.Sprintf("Unanimous approval from %d reviewers", len(votes))
	}

	// Any rejection fails immediately
	if rejectCount > 0 {
		return false, fmt.Sprintf("Rejected by %d reviewer(s), approved by %d, changes requested by %d", rejectCount, approveCount, requestChangeCount)
	}

	// Changes requested
	return false, fmt.Sprintf("Changes requested by %d reviewer(s), approved by %d", requestChangeCount, approveCount)
}

// consolidateFeedback combines feedback from all reviewers
func consolidateFeedback(votes []ReviewVote) string {
	if len(votes) == 0 {
		return "No feedback available"
	}

	feedback := ""
	for i, vote := range votes {
		if vote.Vote == VoteApprove {
			continue // Skip approved reviews
		}

		feedback += fmt.Sprintf("\n## Reviewer %d (%s): %s\n\n%s\n",
			i+1, vote.ReviewType, vote.Vote, vote.Feedback)
	}

	if feedback == "" {
		return "All reviewers approved with no feedback"
	}

	return feedback
}
