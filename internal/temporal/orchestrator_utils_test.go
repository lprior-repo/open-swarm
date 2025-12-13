// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"strings"
	"testing"
)

// TestVoteParser_ParseVote tests vote extraction from reviewer output
func TestVoteParser_ParseVote(t *testing.T) {
	parser := NewVoteParser()

	tests := []struct {
		name     string
		output   string
		wantVote VoteResult
		wantFind bool
	}{
		{
			name:     "explicit approve",
			output:   "The code looks good. APPROVE",
			wantVote: VoteApprove,
			wantFind: true,
		},
		{
			name:     "explicit reject",
			output:   "This has fundamental issues. REJECT",
			wantVote: VoteReject,
			wantFind: true,
		},
		{
			name:     "request change",
			output:   "Please fix the error handling. REQUEST_CHANGE",
			wantVote: VoteRequestChange,
			wantFind: true,
		},
		{
			name:     "case insensitive approve",
			output:   "looks great. approve",
			wantVote: VoteApprove,
			wantFind: true,
		},
		{
			name:     "no explicit vote",
			output:   "This needs some work",
			wantVote: VoteRequestChange,
			wantFind: false,
		},
		{
			name:     "reject takes precedence over approve",
			output:   "I'd normally APPROVE, but there's a critical bug. REJECT",
			wantVote: VoteReject,
			wantFind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseVote(tt.output)
			if result.Vote != tt.wantVote {
				t.Errorf("ParseVote() vote = %v, want %v", result.Vote, tt.wantVote)
			}
			if result.Found != tt.wantFind {
				t.Errorf("ParseVote() found = %v, want %v", result.Found, tt.wantFind)
			}
		})
	}
}

// TestVoteParser_CheckUnanimousApproval tests unanimous approval checking
func TestVoteParser_CheckUnanimousApproval(t *testing.T) {
	parser := NewVoteParser()

	tests := []struct {
		name  string
		votes []ReviewVote
		want  bool
	}{
		{
			name: "all approve",
			votes: []ReviewVote{
				{Vote: VoteApprove, ReviewerName: "r1"},
				{Vote: VoteApprove, ReviewerName: "r2"},
				{Vote: VoteApprove, ReviewerName: "r3"},
			},
			want: true,
		},
		{
			name: "one reject",
			votes: []ReviewVote{
				{Vote: VoteApprove, ReviewerName: "r1"},
				{Vote: VoteReject, ReviewerName: "r2"},
				{Vote: VoteApprove, ReviewerName: "r3"},
			},
			want: false,
		},
		{
			name: "one request change",
			votes: []ReviewVote{
				{Vote: VoteApprove, ReviewerName: "r1"},
				{Vote: VoteRequestChange, ReviewerName: "r2"},
				{Vote: VoteApprove, ReviewerName: "r3"},
			},
			want: false,
		},
		{
			name:  "empty votes",
			votes: []ReviewVote{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parser.CheckUnanimousApproval(tt.votes); got != tt.want {
				t.Errorf("CheckUnanimousApproval() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLintParser_ParseGolangciLint tests lint output parsing
func TestLintParser_ParseGolangciLint(t *testing.T) {
	parser := NewLintParser()

	tests := []struct {
		name       string
		output     string
		wantErrors bool
		wantCount  int
	}{
		{
			name:       "no errors",
			output:     "",
			wantErrors: false,
			wantCount:  0,
		},
		{
			name:       "single error",
			output:     "main.go:10:2: undeclared name: foo (typecheck)",
			wantErrors: true,
			wantCount:  1,
		},
		{
			name: "multiple errors",
			output: `main.go:10:2: undeclared name: foo (typecheck)
main.go:15:5: unused variable bar (unused)
utils.go:20:1: exported function missing comment (golint)`,
			wantErrors: true,
			wantCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseGolangciLint(tt.output)
			if result.HasErrors != tt.wantErrors {
				t.Errorf("ParseGolangciLint() hasErrors = %v, want %v", result.HasErrors, tt.wantErrors)
			}
			if len(result.Issues) != tt.wantCount {
				t.Errorf("ParseGolangciLint() issue count = %d, want %d", len(result.Issues), tt.wantCount)
			}

			// Verify summary is non-empty
			if result.Summary == "" {
				t.Error("ParseGolangciLint() summary should not be empty")
			}
		})
	}
}

// TestLintParser_FormatLintSummary tests lint summary formatting
func TestLintParser_FormatLintSummary(t *testing.T) {
	parser := NewLintParser()

	tests := []struct {
		name   string
		issues []LintIssue
		want   []string // strings that should appear in summary
	}{
		{
			name:   "no issues",
			issues: []LintIssue{},
			want:   []string{"No linting issues"},
		},
		{
			name: "single issue",
			issues: []LintIssue{
				{
					File:    "main.go",
					Line:    10,
					Column:  2,
					Message: "undeclared name: foo",
					Rule:    "typecheck",
				},
			},
			want: []string{"1 linting issue", "main.go", "Line 10", "undeclared name: foo", "typecheck"},
		},
		{
			name: "multiple issues",
			issues: []LintIssue{
				{File: "main.go", Line: 10, Message: "error 1", Rule: "rule1"},
				{File: "main.go", Line: 15, Message: "error 2", Rule: "rule2"},
				{File: "utils.go", Line: 5, Message: "error 3", Rule: "rule3"},
			},
			want: []string{"3 linting issue", "main.go", "utils.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := parser.FormatLintSummary(tt.issues)
			for _, wantStr := range tt.want {
				if !strings.Contains(summary, wantStr) {
					t.Errorf("FormatLintSummary() missing expected string %q in:\n%s", wantStr, summary)
				}
			}
		})
	}
}

// TestReviewAggregator_AggregateReviewFeedback tests review feedback aggregation
func TestReviewAggregator_AggregateReviewFeedback(t *testing.T) {
	aggregator := NewReviewAggregator()

	tests := []struct {
		name  string
		votes []ReviewVote
		want  []string // strings that should appear in aggregated feedback
	}{
		{
			name: "unanimous approval",
			votes: []ReviewVote{
				{Vote: VoteApprove, ReviewType: ReviewTypeTesting},
				{Vote: VoteApprove, ReviewType: ReviewTypeFunctional},
			},
			want: []string{}, // No feedback needed for unanimous approval
		},
		{
			name: "one reject",
			votes: []ReviewVote{
				{Vote: VoteApprove, ReviewType: ReviewTypeTesting, ReviewerName: "r1"},
				{Vote: VoteReject, ReviewType: ReviewTypeFunctional, ReviewerName: "r2", Feedback: "Critical bug in validation logic"},
			},
			want: []string{"Code review feedback", "functional", "REJECT", "Critical bug"},
		},
		{
			name: "mixed feedback",
			votes: []ReviewVote{
				{Vote: VoteRequestChange, ReviewType: ReviewTypeTesting, ReviewerName: "r1", Feedback: "Missing edge case tests"},
				{Vote: VoteReject, ReviewType: ReviewTypeArchitecture, ReviewerName: "r2", Feedback: "Poor separation of concerns"},
			},
			want: []string{"Code review feedback", "testing", "REQUEST_CHANGE", "architecture", "REJECT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedback := aggregator.AggregateReviewFeedback(tt.votes)

			if len(tt.want) == 0 {
				// Unanimous approval should produce empty feedback
				if feedback != "" {
					t.Errorf("AggregateReviewFeedback() expected empty for unanimous approval, got: %s", feedback)
				}
				return
			}

			for _, wantStr := range tt.want {
				if !strings.Contains(strings.ToLower(feedback), strings.ToLower(wantStr)) {
					t.Errorf("AggregateReviewFeedback() missing expected string %q in:\n%s", wantStr, feedback)
				}
			}
		})
	}
}

// TestReviewAggregator_GetRejectionSummary tests rejection summary generation
func TestReviewAggregator_GetRejectionSummary(t *testing.T) {
	aggregator := NewReviewAggregator()

	tests := []struct {
		name  string
		votes []ReviewVote
		want  string
	}{
		{
			name: "all approved",
			votes: []ReviewVote{
				{Vote: VoteApprove},
				{Vote: VoteApprove},
			},
			want: "All reviewers approved",
		},
		{
			name: "one reject",
			votes: []ReviewVote{
				{Vote: VoteApprove},
				{Vote: VoteReject},
			},
			want: "1 reviewer(s) rejected",
		},
		{
			name: "multiple rejects and changes",
			votes: []ReviewVote{
				{Vote: VoteReject},
				{Vote: VoteReject},
				{Vote: VoteRequestChange},
			},
			want: "2 reviewer(s) rejected, 1 requested changes",
		},
		{
			name: "only request changes",
			votes: []ReviewVote{
				{Vote: VoteRequestChange},
				{Vote: VoteRequestChange},
			},
			want: "2 reviewer(s) requested changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := aggregator.GetRejectionSummary(tt.votes)
			if !strings.Contains(summary, tt.want) {
				t.Errorf("GetRejectionSummary() = %q, want to contain %q", summary, tt.want)
			}
		})
	}
}
