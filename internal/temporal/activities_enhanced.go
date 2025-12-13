// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
	"open-swarm/internal/filelock"
)

// EnhancedActivities contains activities for Enhanced TCR workflow
type EnhancedActivities struct {
	lockRegistry *filelock.MemoryRegistry
}

// NewEnhancedActivities creates a new EnhancedActivities instance
func NewEnhancedActivities() *EnhancedActivities {
	return &EnhancedActivities{
		lockRegistry: GetFileLockRegistry(),
	}
}

// AcquireFileLocks acquires locks on task-related files
// Returns list of locked file patterns
func (ea *EnhancedActivities) AcquireFileLocks(ctx context.Context, cellID string, taskID string) ([]string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Acquiring file locks", "cellID", cellID, "taskID", taskID)

	// Define file patterns to lock (task-specific patterns)
	// In a real implementation, these would be derived from task analysis
	patterns := []string{
		fmt.Sprintf("**/%s_test.go", taskID),
		fmt.Sprintf("**/%s.go", taskID),
		"**/test/**",
	}

	lockedPatterns := []string{}
	ttl := 15 * time.Minute // Lock TTL

	for _, pattern := range patterns {
		req := filelock.LockRequest{
			Path:      pattern,
			Holder:    cellID,
			Exclusive: true,
			TTL:       ttl,
		}
		result, err := ea.lockRegistry.Acquire(req)
		if err != nil || !result.Granted {
			// Rollback: release already acquired locks
			for _, locked := range lockedPatterns {
				_ = ea.lockRegistry.Release(locked, cellID)
			}
			return nil, fmt.Errorf("failed to acquire lock on %s: %w", pattern, err)
		}
		lockedPatterns = append(lockedPatterns, pattern)
		logger.Info("Lock acquired", "pattern", pattern, "holder", cellID)
	}

	return lockedPatterns, nil
}

// ReleaseFileLocks releases all locks held by a cell
func (ea *EnhancedActivities) ReleaseFileLocks(ctx context.Context, cellID string, patterns []string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing file locks", "cellID", cellID, "count", len(patterns))

	var errs []error
	for _, pattern := range patterns {
		err := ea.lockRegistry.Release(pattern, cellID)
		if err != nil {
			logger.Warn("Failed to release lock", "pattern", pattern, "error", err)
			errs = append(errs, err)
		} else {
			logger.Info("Lock released", "pattern", pattern)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to release some locks: %v", errs)
	}
	return nil
}

// ExecuteGenTest - Gate 1: Generate test files based on acceptance criteria
func (ea *EnhancedActivities) ExecuteGenTest(ctx context.Context, bootstrap *BootstrapOutput, taskID string, acceptanceCriteria string) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: GenTest", "taskID", taskID)

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Execute agent to generate tests
	prompt := fmt.Sprintf(`Generate comprehensive tests for task: %s

Acceptance Criteria:
%s

Requirements:
- Write tests in Go using testing package
- Cover all edge cases and error conditions
- Follow TDD principles - tests should fail initially
- Use table-driven tests where appropriate`, taskID, acceptanceCriteria)

	result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: fmt.Sprintf("GenTest: %s", taskID),
		Agent: "build",
	})

	if err != nil {
		return &GateResult{
			GateName: "gen_test",
			Passed:   false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	// Get modified files
	fileStatus, _ := cell.Client.GetFileStatus(ctx)
	filesChanged := []string{}
	for _, file := range fileStatus {
		if file.Path != "" {
			filesChanged = append(filesChanged, file.Path)
		}
	}

	return &GateResult{
		GateName: "gen_test",
		Passed:   true,
		AgentResults: []AgentResult{
			{
				AgentName:    "build",
				Model:        "claude-sonnet-4.5",
				Prompt:       prompt,
				Response:     result.GetText(),
				Success:      true,
				Duration:     time.Since(startTime),
				FilesChanged: filesChanged,
			},
		},
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteLintTest - Gate 2: Lint test files
func (ea *EnhancedActivities) ExecuteLintTest(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: LintTest")

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run golangci-lint on test files
	result, err := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"golangci-lint", "run", "--disable-all", "--enable=errcheck,staticcheck,unused", "*_test.go"})

	lintPassed := err == nil
	output := ""
	if result != nil {
		output = result.GetText()
	}

	return &GateResult{
		GateName: "lint_test",
		Passed:   lintPassed,
		LintResult: &LintResult{
			Passed:   lintPassed,
			Output:   output,
			Duration: time.Since(startTime),
			Issues:   []LintIssue{}, // Parse output for detailed issues if needed
		},
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteVerifyRED - Gate 3: Verify tests fail (RED phase)
func (ea *EnhancedActivities) ExecuteVerifyRED(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: VerifyRED")

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run tests - they SHOULD fail
	result, _ := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"go", "test", "-v", "./..."})
	output := ""
	if result != nil {
		output = result.GetText()
	}

	// Parse test output using TestParser
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	// Tests should FAIL - if they pass, that's an error
	testsFailed := parseResult.HasFailures
	redPassed := testsFailed // RED means tests failed as expected

	// Build debug info with parsed failure information
	debugInfo := ""
	if parseResult.HasFailures {
		debugInfo = parser.GetFailureSummary(parseResult)
	}

	return &GateResult{
		GateName: "verify_red",
		Passed:   redPassed,
		TestResult: &TestResult{
			Passed:      !testsFailed, // Inverted: we want false here
			Output:      output,
			Duration:    time.Since(startTime),
			TotalTests:  parseResult.TotalTests,
			FailedTests: parseResult.FailedTests,
		},
		Duration: time.Since(startTime),
		Error: func() string {
			if !redPassed {
				return "tests passed but should fail (not RED)"
			} else {
				return ""
			}
		}(),
		Message: debugInfo,
	}, nil
}

// ExecuteGenImpl - Gate 4: Generate implementation
// testFailureOutput: Optional. If provided (non-empty), includes test failure feedback for retry attempts.
//
// When retrying after VerifyGREEN failures, the workflow should:
//  1. Extract test output from verifyGreenResult.TestResult.Output
//  2. Pass it as testFailureOutput parameter
//  3. The agent will receive parsed, structured failure information
//
// Example retry workflow:
//
//	if !verifyGreenResult.Passed {
//	  testOutput := verifyGreenResult.TestResult.Output
//	  genImplResult, _ := ExecuteGenImpl(ctx, bootstrap, taskID, desc, criteria, testOutput)
//	}
func (ea *EnhancedActivities) ExecuteGenImpl(ctx context.Context, bootstrap *BootstrapOutput, taskID string, description string, acceptanceCriteria string, testFailureOutput string) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: GenImpl", "taskID", taskID)

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Build base prompt
	var promptBuilder strings.Builder

	// If retry feedback is provided, include parsed test failures
	if testFailureOutput != "" {
		parser := NewTestParser()
		parseResult := parser.ParseTestOutput(testFailureOutput)

		if parseResult.HasFailures {
			failureSummary := parser.GetFailureSummary(parseResult)
			promptBuilder.WriteString("Previous implementation attempt failed with test failures:\n\n")
			promptBuilder.WriteString(failureSummary)
			promptBuilder.WriteString("\n\nPlease fix the implementation to address these failures.\n\n")
			logger.Info("GenImpl retry with test failure feedback", "failures", len(parseResult.Failures))
		}
	}

	promptBuilder.WriteString(fmt.Sprintf(`Implement the solution for task: %s

Description: %s

Acceptance Criteria:
%s

Requirements:
- Implement code to make all tests pass
- Follow Go best practices and idioms
- Include proper error handling
- Add documentation comments`, taskID, description, acceptanceCriteria))

	prompt := promptBuilder.String()

	result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: fmt.Sprintf("GenImpl: %s", taskID),
		Agent: "build",
	})

	if err != nil {
		return &GateResult{
			GateName: "gen_impl",
			Passed:   false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	// Get modified files
	fileStatus, _ := cell.Client.GetFileStatus(ctx)
	filesChanged := []string{}
	for _, file := range fileStatus {
		if file.Path != "" {
			filesChanged = append(filesChanged, file.Path)
		}
	}

	return &GateResult{
		GateName: "gen_impl",
		Passed:   true,
		AgentResults: []AgentResult{
			{
				AgentName:    "build",
				Model:        "claude-sonnet-4.5",
				Prompt:       prompt,
				Response:     result.GetText(),
				Success:      true,
				Duration:     time.Since(startTime),
				FilesChanged: filesChanged,
			},
		},
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteVerifyGREEN - Gate 5: Verify tests pass (GREEN phase)
func (ea *EnhancedActivities) ExecuteVerifyGREEN(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: VerifyGREEN")

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run tests - they SHOULD pass
	result, err := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"go", "test", "-v", "./..."})
	output := ""
	if result != nil {
		output = result.GetText()
	}

	// Parse test output using TestParser
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	// GREEN means all tests pass - no failures allowed
	testsPassed := err == nil && !parseResult.HasFailures

	// Extract failed test names for detailed reporting
	failedTestNames := make([]string, 0, len(parseResult.Failures))
	for _, failure := range parseResult.Failures {
		failedTestNames = append(failedTestNames, failure.TestName)
	}

	return &GateResult{
		GateName: "verify_green",
		Passed:   testsPassed,
		TestResult: &TestResult{
			Passed:       testsPassed,
			TotalTests:   parseResult.TotalTests,
			PassedTests:  parseResult.PassedTests,
			FailedTests:  parseResult.FailedTests,
			Output:       output,
			Duration:     time.Since(startTime),
			FailureTests: failedTestNames,
		},
		Duration: time.Since(startTime),
		Error: func() string {
			if !testsPassed {
				if len(parseResult.Failures) > 0 {
					return fmt.Sprintf("tests failed (not GREEN): %d failure(s) detected", len(parseResult.Failures))
				}
				return "tests failed (not GREEN)"
			} else {
				return ""
			}
		}(),
	}, nil
}

// ExecuteMultiReview - Gate 6: Multi-reviewer approval (3 reviewers, unanimous)
func (ea *EnhancedActivities) ExecuteMultiReview(ctx context.Context, bootstrap *BootstrapOutput, taskID string, description string, reviewersCount int) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Gate: MultiReview", "reviewers", reviewersCount)

	startTime := time.Now()
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	reviewTypes := []ReviewType{ReviewTypeTesting, ReviewTypeFunctional, ReviewTypeArchitecture}
	votes := []ReviewVote{}
	allApproved := true

	for i := 0; i < reviewersCount; i++ {
		reviewType := reviewTypes[i%len(reviewTypes)]
		reviewStart := time.Now()

		prompt := fmt.Sprintf(`Review the code changes for task: %s

Description: %s

Review Focus: %s

Provide:
1. Vote: APPROVE, REQUEST_CHANGE, or REJECT
2. Detailed feedback on the implementation
3. Specific issues or improvements needed

Your review should focus on: %s`, taskID, description, reviewType, getReviewFocus(reviewType))

		result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
			Title: fmt.Sprintf("Review %d (%s): %s", i+1, reviewType, taskID),
			Agent: "build",
		})

		vote := VoteApprove // Default
		feedback := ""

		if err != nil {
			vote = VoteReject
			feedback = fmt.Sprintf("Review failed: %v", err)
			allApproved = false
		} else {
			feedback = result.GetText()
			// Simple heuristic: check for rejection keywords
			feedbackLower := strings.ToLower(feedback)
			if strings.Contains(feedbackLower, "reject") {
				vote = VoteReject
				allApproved = false
			} else if strings.Contains(feedbackLower, "request") || strings.Contains(feedbackLower, "change") {
				vote = VoteRequestChange
				allApproved = false
			}
		}

		votes = append(votes, ReviewVote{
			ReviewerName: fmt.Sprintf("reviewer-%d", i+1),
			ReviewType:   reviewType,
			Vote:         vote,
			Feedback:     feedback,
			Duration:     time.Since(reviewStart),
		})
	}

	return &GateResult{
		GateName:    "multi_review",
		Passed:      allApproved,
		ReviewVotes: votes,
		Duration:    time.Since(startTime),
		Error: func() string {
			if !allApproved {
				return "not all reviewers approved"
			} else {
				return ""
			}
		}(),
	}, nil
}

// GetCommitSHA retrieves the current commit SHA
func (ea *EnhancedActivities) GetCommitSHA(ctx context.Context, bootstrap *BootstrapOutput) (string, error) {
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	result, err := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"git", "rev-parse", "HEAD"})
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(result.GetText())
	return sha, nil
}

// Helper function to get review focus description
func getReviewFocus(reviewType ReviewType) string {
	switch reviewType {
	case ReviewTypeTesting:
		return "test coverage, quality, edge cases, and test maintainability"
	case ReviewTypeFunctional:
		return "correctness, requirements satisfaction, and behavior"
	case ReviewTypeArchitecture:
		return "design patterns, code structure, and long-term maintainability"
	default:
		return "overall code quality"
	}
}
