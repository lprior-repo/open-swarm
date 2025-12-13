// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"regexp"
	"strings"
)

// VoteParser extracts structured review votes from LLM output
type VoteParser struct{}

// NewVoteParser creates a new vote parser
func NewVoteParser() *VoteParser {
	return &VoteParser{}
}

// ParsedVote contains extracted vote information
type ParsedVote struct {
	Vote     VoteResult
	Feedback string
	Found    bool
}

// ParseVote extracts vote decision from reviewer output
// Looks for explicit vote markers: APPROVE, REQUEST_CHANGE, REJECT
// Returns the most specific/severe vote found
func (vp *VoteParser) ParseVote(output string) ParsedVote {
	// Vote patterns in order of severity (most to least severe)
	patterns := []struct {
		vote    VoteResult
		pattern *regexp.Regexp
	}{
		{VoteReject, regexp.MustCompile(`(?i)\bREJECT\b`)},
		{VoteRequestChange, regexp.MustCompile(`(?i)\bREQUEST[_ ]CHANGE\b`)},
		{VoteApprove, regexp.MustCompile(`(?i)\bAPPROVE\b`)},
	}

	for _, p := range patterns {
		if p.pattern.MatchString(output) {
			return ParsedVote{
				Vote:     p.vote,
				Feedback: output,
				Found:    true,
			}
		}
	}

	// Default to REQUEST_CHANGE if no explicit vote found
	return ParsedVote{
		Vote:     VoteRequestChange,
		Feedback: output,
		Found:    false,
	}
}

// CheckUnanimousApproval verifies all votes are APPROVE
func (vp *VoteParser) CheckUnanimousApproval(votes []ReviewVote) bool {
	if len(votes) == 0 {
		return false
	}

	for _, vote := range votes {
		if vote.Vote != VoteApprove {
			return false
		}
	}
	return true
}

// LintParser formats linting output into structured feedback
type LintParser struct{}

// NewLintParser creates a new lint parser
func NewLintParser() *LintParser {
	return &LintParser{}
}

// ParsedLintResult contains structured lint information
type ParsedLintResult struct {
	HasErrors bool
	Issues    []LintIssue
	Summary   string
}

// ParseGolangciLint parses golangci-lint output into structured issues
func (lp *LintParser) ParseGolangciLint(output string) ParsedLintResult {
	if strings.TrimSpace(output) == "" {
		return ParsedLintResult{
			HasErrors: false,
			Issues:    []LintIssue{},
			Summary:   "No linting issues found",
		}
	}

	// golangci-lint format: file:line:col: message (rule)
	// Example: main.go:10:2: undeclared name: foo (typecheck)
	pattern := regexp.MustCompile(`([^:]+):(\d+):(\d+):\s*(.+?)\s*\(([^)]+)\)`)

	lines := strings.Split(output, "\n")
	issues := []LintIssue{}

	for _, line := range lines {
		matches := pattern.FindStringSubmatch(line)
		if matches != nil && len(matches) >= 6 {
			// Extract line and column numbers
			lineNum := 0
			colNum := 0
			fmt.Sscanf(matches[2], "%d", &lineNum)
			fmt.Sscanf(matches[3], "%d", &colNum)

			issue := LintIssue{
				File:     matches[1],
				Line:     lineNum,
				Column:   colNum,
				Message:  matches[4],
				Rule:     matches[5],
				Severity: "error", // golangci-lint defaults to error
			}
			issues = append(issues, issue)
		}
	}

	hasErrors := len(issues) > 0
	summary := lp.FormatLintSummary(issues)

	return ParsedLintResult{
		HasErrors: hasErrors,
		Issues:    issues,
		Summary:   summary,
	}
}

// FormatLintSummary creates agent-friendly feedback for lint issues
func (lp *LintParser) FormatLintSummary(issues []LintIssue) string {
	if len(issues) == 0 {
		return "No linting issues found"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d linting issue(s):\n\n", len(issues)))

	// Group by file for better readability
	fileIssues := make(map[string][]LintIssue)
	for _, issue := range issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	for file, fileIssueList := range fileIssues {
		builder.WriteString(fmt.Sprintf("File: %s\n", file))
		for _, issue := range fileIssueList {
			builder.WriteString(fmt.Sprintf("  Line %d, Col %d: %s [%s]\n",
				issue.Line, issue.Column, issue.Message, issue.Rule))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// ReviewAggregator combines multiple reviewer feedback into coherent retry prompt
type ReviewAggregator struct {
	voteParser *VoteParser
}

// NewReviewAggregator creates a new review aggregator
func NewReviewAggregator() *ReviewAggregator {
	return &ReviewAggregator{
		voteParser: NewVoteParser(),
	}
}

// AggregateReviewFeedback combines rejection/change-request feedback from reviewers
func (ra *ReviewAggregator) AggregateReviewFeedback(votes []ReviewVote) string {
	if len(votes) == 0 {
		return ""
	}

	// Check if unanimous approval (no feedback needed)
	if ra.voteParser.CheckUnanimousApproval(votes) {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Code review feedback requiring changes:\n\n")

	// Group by review type for clarity
	byType := make(map[ReviewType][]ReviewVote)
	for _, vote := range votes {
		if vote.Vote != VoteApprove {
			byType[vote.ReviewType] = append(byType[vote.ReviewType], vote)
		}
	}

	// Format feedback by type
	reviewTypes := []ReviewType{ReviewTypeTesting, ReviewTypeFunctional, ReviewTypeArchitecture}
	for _, reviewType := range reviewTypes {
		typeVotes := byType[reviewType]
		if len(typeVotes) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("%s Review:\n", reviewType))
		for _, vote := range typeVotes {
			builder.WriteString(fmt.Sprintf("  [%s] %s\n", vote.Vote, vote.ReviewerName))
			// Extract key points from feedback (first 3 lines for conciseness)
			feedbackLines := strings.Split(vote.Feedback, "\n")
			maxLines := 3
			if len(feedbackLines) < maxLines {
				maxLines = len(feedbackLines)
			}
			for i := 0; i < maxLines; i++ {
				if strings.TrimSpace(feedbackLines[i]) != "" {
					builder.WriteString(fmt.Sprintf("    %s\n", feedbackLines[i]))
				}
			}
		}
		builder.WriteString("\n")
	}

	builder.WriteString("Please address the above concerns in your next implementation attempt.\n")

	return builder.String()
}

// GetRejectionSummary creates a concise summary of why reviews were rejected
func (ra *ReviewAggregator) GetRejectionSummary(votes []ReviewVote) string {
	rejects := 0
	requestChanges := 0

	for _, vote := range votes {
		switch vote.Vote {
		case VoteReject:
			rejects++
		case VoteRequestChange:
			requestChanges++
		}
	}

	if rejects > 0 {
		return fmt.Sprintf("%d reviewer(s) rejected, %d requested changes", rejects, requestChanges)
	}
	if requestChanges > 0 {
		return fmt.Sprintf("%d reviewer(s) requested changes", requestChanges)
	}
	return "All reviewers approved"
}
