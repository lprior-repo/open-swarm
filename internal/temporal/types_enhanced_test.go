// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"encoding/json"
	"fmt"
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
	// This test is now replaced by more comprehensive tests below
	// TestEnhancedTCRResult_Fields, TestEnhancedTCRResult_SuccessfulWorkflow, etc.
	result := EnhancedTCRResult{
		Success: true,
		GateResults: []GateResult{
			{
				GateName: "bootstrap",
				Passed:   true,
				Duration: 5 * time.Second,
			},
		},
		FilesChanged: []string{"auth.go", "auth_test.go"},
		Error:        "",
	}

	assert.True(t, result.Success)
	assert.Len(t, result.GateResults, 1)
	assert.Equal(t, "bootstrap", result.GateResults[0].GateName)
	assert.Len(t, result.FilesChanged, 2)
	assert.Empty(t, result.Error)
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

func TestEnhancedTCRResult_Fields(t *testing.T) {
	// Test basic field assignment
	result := EnhancedTCRResult{
		Success: true,
		GateResults: []GateResult{
			{
				GateName: "bootstrap",
				Passed:   true,
				Duration: 5 * time.Second,
			},
			{
				GateName: "verify_green",
				Passed:   true,
				Duration: 30 * time.Second,
			},
		},
		FilesChanged: []string{"auth.go", "auth_test.go", "utils.go"},
		Error:        "",
	}

	assert.True(t, result.Success)
	assert.Len(t, result.GateResults, 2)
	assert.Equal(t, "bootstrap", result.GateResults[0].GateName)
	assert.Equal(t, "verify_green", result.GateResults[1].GateName)
	assert.Len(t, result.FilesChanged, 3)
	assert.Equal(t, "auth.go", result.FilesChanged[0])
	assert.Equal(t, "auth_test.go", result.FilesChanged[1])
	assert.Equal(t, "utils.go", result.FilesChanged[2])
	assert.Empty(t, result.Error)
}

func TestEnhancedTCRResult_EmptyArrays(t *testing.T) {
	// Test with empty arrays
	result := EnhancedTCRResult{
		Success:      false,
		GateResults:  []GateResult{},
		FilesChanged: []string{},
		Error:        "workflow initialization failed",
	}

	assert.False(t, result.Success)
	assert.Empty(t, result.GateResults)
	assert.Empty(t, result.FilesChanged)
	assert.Equal(t, "workflow initialization failed", result.Error)
}

func TestEnhancedTCRResult_NilArrays(t *testing.T) {
	// Test with nil arrays (should be allowed)
	result := EnhancedTCRResult{
		Success:      false,
		GateResults:  nil,
		FilesChanged: nil,
		Error:        "bootstrap failed",
	}

	assert.False(t, result.Success)
	assert.Nil(t, result.GateResults)
	assert.Nil(t, result.FilesChanged)
	assert.Equal(t, "bootstrap failed", result.Error)
}

func TestEnhancedTCRResult_FailureScenario(t *testing.T) {
	// Test failure scenario with gate results showing where it failed
	result := EnhancedTCRResult{
		Success: false,
		GateResults: []GateResult{
			{
				GateName: "bootstrap",
				Passed:   true,
				Duration: 3 * time.Second,
			},
			{
				GateName: "gen_test",
				Passed:   true,
				Duration: 45 * time.Second,
			},
			{
				GateName: "verify_red",
				Passed:   false,
				Duration: 10 * time.Second,
				Error:    "test did not fail as expected",
			},
		},
		FilesChanged: []string{"handler_test.go"},
		Error:        "verify_red gate failed: test did not fail as expected",
	}

	assert.False(t, result.Success)
	assert.Len(t, result.GateResults, 3)
	assert.True(t, result.GateResults[0].Passed)
	assert.True(t, result.GateResults[1].Passed)
	assert.False(t, result.GateResults[2].Passed)
	assert.Equal(t, "verify_red", result.GateResults[2].GateName)
	assert.Contains(t, result.Error, "verify_red gate failed")
}

func TestEnhancedTCRResult_SuccessfulWorkflow(t *testing.T) {
	// Test complete successful workflow with all gates
	result := EnhancedTCRResult{
		Success: true,
		GateResults: []GateResult{
			{GateName: "bootstrap", Passed: true, Duration: 2 * time.Second},
			{GateName: "gen_test", Passed: true, Duration: 40 * time.Second},
			{GateName: "lint_test", Passed: true, Duration: 5 * time.Second},
			{GateName: "verify_red", Passed: true, Duration: 8 * time.Second},
			{GateName: "gen_impl", Passed: true, Duration: 60 * time.Second},
			{GateName: "verify_green", Passed: true, Duration: 10 * time.Second},
		},
		FilesChanged: []string{
			"internal/api/handler.go",
			"internal/api/handler_test.go",
		},
		Error: "",
	}

	assert.True(t, result.Success)
	assert.Len(t, result.GateResults, 6)

	// Verify all gates passed
	for _, gate := range result.GateResults {
		assert.True(t, gate.Passed, "gate %s should have passed", gate.GateName)
	}

	assert.Len(t, result.FilesChanged, 2)
	assert.Empty(t, result.Error)
}

func TestEnhancedTCRResult_JSONSerialization(t *testing.T) {
	// Test JSON marshaling and unmarshaling (required for Temporal)
	original := EnhancedTCRResult{
		Success: true,
		GateResults: []GateResult{
			{
				GateName: "bootstrap",
				Passed:   true,
				Duration: 5 * time.Second,
			},
		},
		FilesChanged: []string{"main.go", "main_test.go"},
		Error:        "",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded EnhancedTCRResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Success, decoded.Success)
	assert.Len(t, decoded.GateResults, 1)
	assert.Equal(t, original.GateResults[0].GateName, decoded.GateResults[0].GateName)
	assert.Equal(t, original.GateResults[0].Passed, decoded.GateResults[0].Passed)
	assert.Equal(t, original.FilesChanged, decoded.FilesChanged)
	assert.Equal(t, original.Error, decoded.Error)
}

func TestEnhancedTCRResult_ArrayHandling(t *testing.T) {
	t.Run("single gate result", func(t *testing.T) {
		result := EnhancedTCRResult{
			Success: true,
			GateResults: []GateResult{
				{GateName: "bootstrap", Passed: true},
			},
			FilesChanged: []string{"file.go"},
			Error:        "",
		}

		assert.Len(t, result.GateResults, 1)
		assert.Len(t, result.FilesChanged, 1)
	})

	t.Run("multiple gate results", func(t *testing.T) {
		gateResults := make([]GateResult, 10)
		for i := 0; i < 10; i++ {
			gateResults[i] = GateResult{
				GateName: fmt.Sprintf("gate_%d", i),
				Passed:   true,
			}
		}

		result := EnhancedTCRResult{
			Success:      true,
			GateResults:  gateResults,
			FilesChanged: []string{"a.go", "b.go", "c.go"},
			Error:        "",
		}

		assert.Len(t, result.GateResults, 10)
		assert.Equal(t, "gate_0", result.GateResults[0].GateName)
		assert.Equal(t, "gate_9", result.GateResults[9].GateName)
	})

	t.Run("many files changed", func(t *testing.T) {
		files := make([]string, 20)
		for i := 0; i < 20; i++ {
			files[i] = fmt.Sprintf("file_%d.go", i)
		}

		result := EnhancedTCRResult{
			Success:      true,
			GateResults:  []GateResult{{GateName: "test", Passed: true}},
			FilesChanged: files,
			Error:        "",
		}

		assert.Len(t, result.FilesChanged, 20)
		assert.Equal(t, "file_0.go", result.FilesChanged[0])
		assert.Equal(t, "file_19.go", result.FilesChanged[19])
	})
}

func TestEnhancedTCRResult_ZeroValue(t *testing.T) {
	// Test zero value struct
	var result EnhancedTCRResult

	assert.False(t, result.Success)
	assert.Nil(t, result.GateResults)
	assert.Nil(t, result.FilesChanged)
	assert.Empty(t, result.Error)
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
	// This test is now replaced by TestEnhancedTCRResult_JSONSerialization
	// which tests the simplified struct format
	result := EnhancedTCRResult{
		Success: true,
		GateResults: []GateResult{
			{
				GateName: "verify_green",
				Passed:   true,
				Duration: 1 * time.Minute,
			},
		},
		FilesChanged: []string{"main.go"},
		Error:        "",
	}

	// Test JSON serialization (required for Temporal)
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded EnhancedTCRResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Success, decoded.Success)
	assert.Len(t, decoded.GateResults, 1)
	assert.Equal(t, result.GateResults[0].GateName, decoded.GateResults[0].GateName)
	assert.Equal(t, result.FilesChanged, decoded.FilesChanged)
	assert.Equal(t, result.Error, decoded.Error)
}
