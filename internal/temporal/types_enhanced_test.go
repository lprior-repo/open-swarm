// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedTCRInput_Fields(t *testing.T) {
	input := EnhancedTCRInput{
		CellID:             "test-cell-1",
		Branch:             "main",
		TaskID:             "task-123",
		Description:        "Add user validation",
		AcceptanceCriteria: "Must validate email format",
		ReviewersCount:     3,
	}

	assert.Equal(t, "test-cell-1", input.CellID)
	assert.Equal(t, "main", input.Branch)
	assert.Equal(t, "task-123", input.TaskID)
	assert.Equal(t, "Add user validation", input.Description)
	assert.Equal(t, "Must validate email format", input.AcceptanceCriteria)
	assert.Equal(t, 3, input.ReviewersCount)
}

func TestEnhancedTCRInput_Serialization(t *testing.T) {
	input := EnhancedTCRInput{
		CellID:             "cell-1",
		Branch:             "feature/auth",
		TaskID:             "beads-123",
		Description:        "Implement OAuth2",
		AcceptanceCriteria: "Must support Google and GitHub",
		ReviewersCount:     3,
	}

	// Test JSON serialization
	data, err := json.Marshal(input)
	require.NoError(t, err)

	// Test JSON deserialization
	var decoded EnhancedTCRInput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, input, decoded)
}

func TestEnhancedTCRResult(t *testing.T) {
	result := EnhancedTCRResult{
		Success:       true,
		WorkflowState: StateComplete,
		Gates: map[string]GateResult{
			"bootstrap": {
				GateName: "bootstrap",
				Passed:   true,
				Duration: 5 * time.Second,
			},
		},
		FinalCommitSHA: "abc123def456",
		FilesChanged:   []string{"auth.go", "auth_test.go"},
		TotalDuration:  2 * time.Minute,
		RetryCount:     0,
	}

	assert.True(t, result.Success)
	assert.Equal(t, StateComplete, result.WorkflowState)
	assert.Len(t, result.Gates, 1)
	assert.Equal(t, "abc123def456", result.FinalCommitSHA)
	assert.Len(t, result.FilesChanged, 2)
}

func TestWorkflowStates(t *testing.T) {
	states := []WorkflowState{
		StateBootstrap,
		StateGenTest,
		StateLintTest,
		StateVerifyRED,
		StateGenImpl,
		StateVerifyGREEN,
		StateMultiReview,
		StateCommit,
		StateComplete,
		StateFailed,
	}

	expectedStates := []string{
		"bootstrap",
		"gen_test",
		"lint_test",
		"verify_red",
		"gen_impl",
		"verify_green",
		"multi_review",
		"commit",
		"complete",
		"failed",
	}

	for i, state := range states {
		assert.Equal(t, expectedStates[i], string(state))
	}
}

func TestGateResult(t *testing.T) {
	testResult := &TestResult{
		Passed:      true,
		TotalTests:  10,
		PassedTests: 10,
		FailedTests: 0,
		Output:      "All tests passed",
		Duration:    30 * time.Second,
	}

	gate := GateResult{
		GateName: "verify_green",
		Passed:   true,
		AgentResults: []AgentResult{
			{
				AgentName:    "impl-agent",
				Model:        "haiku-4.5",
				Success:      true,
				Duration:     45 * time.Second,
				FilesChanged: []string{"impl.go"},
			},
		},
		Duration:      60 * time.Second,
		TestResult:    testResult,
		RetryAttempts: 0,
	}

	assert.Equal(t, "verify_green", gate.GateName)
	assert.True(t, gate.Passed)
	assert.Len(t, gate.AgentResults, 1)
	assert.NotNil(t, gate.TestResult)
	assert.True(t, gate.TestResult.Passed)
}

func TestTestResult(t *testing.T) {
	t.Run("all tests pass", func(t *testing.T) {
		result := TestResult{
			Passed:      true,
			TotalTests:  15,
			PassedTests: 15,
			FailedTests: 0,
			Output:      "PASS",
			Duration:    20 * time.Second,
		}

		assert.True(t, result.Passed)
		assert.Equal(t, 15, result.TotalTests)
		assert.Equal(t, 0, result.FailedTests)
		assert.Empty(t, result.FailureTests)
	})

	t.Run("some tests fail", func(t *testing.T) {
		result := TestResult{
			Passed:       false,
			TotalTests:   10,
			PassedTests:  7,
			FailedTests:  3,
			Output:       "FAIL",
			Duration:     25 * time.Second,
			FailureTests: []string{"TestAuth", "TestValidation", "TestEdgeCase"},
		}

		assert.False(t, result.Passed)
		assert.Equal(t, 10, result.TotalTests)
		assert.Equal(t, 3, result.FailedTests)
		assert.Len(t, result.FailureTests, 3)
	})
}

func TestLintResult(t *testing.T) {
	t.Run("no lint issues", func(t *testing.T) {
		result := LintResult{
			Passed:   true,
			Issues:   []LintIssue{},
			Output:   "No issues found",
			Duration: 5 * time.Second,
		}

		assert.True(t, result.Passed)
		assert.Empty(t, result.Issues)
	})

	t.Run("with lint issues", func(t *testing.T) {
		result := LintResult{
			Passed: false,
			Issues: []LintIssue{
				{
					File:     "auth.go",
					Line:     42,
					Column:   10,
					Severity: "error",
					Message:  "undefined variable",
					Rule:     "undefined-var",
				},
				{
					File:     "utils.go",
					Line:     15,
					Column:   5,
					Severity: "warning",
					Message:  "unused import",
					Rule:     "unused-import",
				},
			},
			Output:   "2 issues found",
			Duration: 3 * time.Second,
		}

		assert.False(t, result.Passed)
		assert.Len(t, result.Issues, 2)
		assert.Equal(t, "error", result.Issues[0].Severity)
		assert.Equal(t, "warning", result.Issues[1].Severity)
	})
}

func TestReviewVote(t *testing.T) {
	vote := ReviewVote{
		ReviewerName: "testing-reviewer",
		ReviewType:   ReviewTypeTesting,
		Vote:         VoteApprove,
		Feedback:     "Test coverage is excellent, edge cases handled well",
		Duration:     2 * time.Minute,
	}

	assert.Equal(t, "testing-reviewer", vote.ReviewerName)
	assert.Equal(t, ReviewTypeTesting, vote.ReviewType)
	assert.Equal(t, VoteApprove, vote.Vote)
	assert.Contains(t, vote.Feedback, "excellent")
}

func TestReviewTypes(t *testing.T) {
	types := []ReviewType{
		ReviewTypeTesting,
		ReviewTypeFunctional,
		ReviewTypeArchitecture,
	}

	expected := []string{"testing", "functional", "architecture"}

	for i, rt := range types {
		assert.Equal(t, expected[i], string(rt))
	}
}

func TestVoteResults(t *testing.T) {
	votes := []VoteResult{
		VoteApprove,
		VoteRequestChange,
		VoteReject,
	}

	expected := []string{"APPROVE", "REQUEST_CHANGE", "REJECT"}

	for i, vote := range votes {
		assert.Equal(t, expected[i], string(vote))
	}
}

func TestUnanimousVoting(t *testing.T) {
	t.Run("unanimous approval", func(t *testing.T) {
		votes := []ReviewVote{
			{ReviewType: ReviewTypeTesting, Vote: VoteApprove},
			{ReviewType: ReviewTypeFunctional, Vote: VoteApprove},
			{ReviewType: ReviewTypeArchitecture, Vote: VoteApprove},
		}

		allApproved := true
		for _, vote := range votes {
			if vote.Vote != VoteApprove {
				allApproved = false
				break
			}
		}

		assert.True(t, allApproved)
	})

	t.Run("one rejection fails unanimous", func(t *testing.T) {
		votes := []ReviewVote{
			{ReviewType: ReviewTypeTesting, Vote: VoteApprove},
			{ReviewType: ReviewTypeFunctional, Vote: VoteApprove},
			{ReviewType: ReviewTypeArchitecture, Vote: VoteReject},
		}

		allApproved := true
		for _, vote := range votes {
			if vote.Vote != VoteApprove {
				allApproved = false
				break
			}
		}

		assert.False(t, allApproved)
	})
}

func TestEnhancedTCRResult_Serialization(t *testing.T) {
	result := EnhancedTCRResult{
		Success:       true,
		WorkflowState: StateComplete,
		Gates: map[string]GateResult{
			"verify_green": {
				GateName: "verify_green",
				Passed:   true,
				Duration: 1 * time.Minute,
			},
		},
		FinalCommitSHA: "abc123",
		FilesChanged:   []string{"main.go"},
		TotalDuration:  5 * time.Minute,
		RetryCount:     1,
		ReviewVotes: []ReviewVote{
			{ReviewerName: "testing", ReviewType: ReviewTypeTesting, Vote: VoteApprove},
		},
	}

	// Test JSON serialization (required for Temporal)
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded EnhancedTCRResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Success, decoded.Success)
	assert.Equal(t, result.WorkflowState, decoded.WorkflowState)
	assert.Equal(t, result.FinalCommitSHA, decoded.FinalCommitSHA)
	assert.Len(t, decoded.ReviewVotes, 1)
}
