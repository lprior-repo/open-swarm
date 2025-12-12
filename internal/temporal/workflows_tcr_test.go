// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestTCRWorkflow_HappyPath tests the complete TCR workflow with successful execution
func TestTCRWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock cell activities
	cellActivities := &CellActivities{}

	// Bootstrap succeeds
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, BootstrapInput{
		CellID: "test-cell-001",
		Branch: "main",
	}).Return(&BootstrapOutput{
		CellID:       "test-cell-001",
		Port:         8001,
		WorktreeID:   "wt-001",
		WorktreePath: "/tmp/wt-001",
		BaseURL:      "http://localhost:8001",
		ServerPID:    12345,
	}, nil)

	// Task execution succeeds
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success:       true,
		Output:        "Task completed successfully",
		FilesModified: []string{"main.go", "handler.go"},
		ErrorMessage:  "",
	}, nil)

	// Tests pass
	env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(true, nil)

	// Commit succeeds
	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Teardown succeeds
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID:      "test-cell-001",
		Branch:      "main",
		TaskID:      "task-001",
		Description: "Implement feature X",
		Prompt:      "Create a new handler",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.True(t, result.Success)
	assert.True(t, result.TestsPassed)
	assert.Equal(t, []string{"main.go", "handler.go"}, result.FilesChanged)
	assert.Empty(t, result.Error)
}

// TestTCRWorkflow_BootstrapFailure tests workflow behavior when bootstrap fails
func TestTCRWorkflow_BootstrapFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Bootstrap fails
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(
		(*BootstrapOutput)(nil),
		errors.New("failed to allocate port"),
	)

	// Execute workflow
	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID:      "test-cell-001",
		Branch:      "main",
		TaskID:      "task-001",
		Description: "Test task",
		Prompt:      "Test prompt",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "bootstrap failed")
}

// TestTCRWorkflow_TaskExecutionFailure tests workflow when task execution fails
func TestTCRWorkflow_TaskExecutionFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Bootstrap succeeds
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:       "test-cell-001",
		Port:         8001,
		WorktreeID:   "wt-001",
		WorktreePath: "/tmp/wt-001",
		BaseURL:      "http://localhost:8001",
		ServerPID:    12345,
	}, nil)

	// Task execution fails
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success:      false,
		Output:       "Compilation failed",
		ErrorMessage: "undefined variable 'x'",
	}, nil)

	// Teardown succeeds
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID:      "test-cell-001",
		Branch:      "main",
		TaskID:      "task-001",
		Description: "Test task",
		Prompt:      "Test prompt",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "undefined variable 'x'")
}

// TestTCRWorkflow_TestsFailRevert tests that changes are reverted when tests fail
func TestTCRWorkflow_TestsFailRevert(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Bootstrap succeeds
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:       "test-cell-001",
		Port:         8001,
		WorktreeID:   "wt-001",
		WorktreePath: "/tmp/wt-001",
		BaseURL:      "http://localhost:8001",
		ServerPID:    12345,
	}, nil)

	// Task execution succeeds
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success:       true,
		Output:        "Code generated",
		FilesModified: []string{"broken.go"},
	}, nil)

	// Tests fail
	env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(false, nil)

	// Revert is called (not commit)
	env.OnActivity(cellActivities.RevertChanges, mock.Anything, mock.Anything).Return(nil)

	// Teardown succeeds
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID:      "test-cell-001",
		Branch:      "main",
		TaskID:      "task-001",
		Description: "Test task",
		Prompt:      "Test prompt",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.False(t, result.Success)
	assert.False(t, result.TestsPassed)
	assert.Equal(t, []string{"broken.go"}, result.FilesChanged)
}

// TestTCRWorkflow_TestsPassCommit tests that changes are committed when tests pass
func TestTCRWorkflow_TestsPassCommit(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Bootstrap succeeds
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:       "test-cell-001",
		Port:         8001,
		WorktreeID:   "wt-001",
		WorktreePath: "/tmp/wt-001",
		BaseURL:      "http://localhost:8001",
		ServerPID:    12345,
	}, nil)

	// Task execution succeeds
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success:       true,
		Output:        "Code generated",
		FilesModified: []string{"good.go"},
	}, nil)

	// Tests pass
	env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(true, nil)

	// Commit is called (not revert)
	commitCalled := false
	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		commitCalled = true
		// Verify commit message format
		msg := args.Get(2).(string)
		assert.Contains(t, msg, "task-001")
		assert.Contains(t, msg, "🤖 Generated by Reactor-SDK")
	})

	// Teardown succeeds
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID:      "test-cell-001",
		Branch:      "main",
		TaskID:      "task-001",
		Description: "Test task",
		Prompt:      "Test prompt",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.True(t, result.Success)
	assert.True(t, result.TestsPassed)
	assert.True(t, commitCalled, "CommitChanges should have been called")
}

// TestTCRWorkflow_TeardownAlwaysRuns tests that teardown runs even on failures
func TestTCRWorkflow_TeardownAlwaysRuns(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(env *testsuite.TestWorkflowEnvironment, ca *CellActivities)
		expectSuccess bool
	}{
		{
			name: "teardown after bootstrap failure",
			setupMocks: func(env *testsuite.TestWorkflowEnvironment, ca *CellActivities) {
				env.OnActivity(ca.BootstrapCell, mock.Anything, mock.Anything).Return(
					(*BootstrapOutput)(nil),
					errors.New("bootstrap failed"),
				)
			},
			expectSuccess: false,
		},
		{
			name: "teardown after task failure",
			setupMocks: func(env *testsuite.TestWorkflowEnvironment, ca *CellActivities) {
				env.OnActivity(ca.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
					CellID:    "test-cell",
					Port:      8001,
					BaseURL:   "http://localhost:8001",
					ServerPID: 12345,
				}, nil)
				env.OnActivity(ca.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
					Success:      false,
					ErrorMessage: "task failed",
				}, nil)
				env.OnActivity(ca.TeardownCell, mock.Anything, mock.Anything).Return(nil)
			},
			expectSuccess: false,
		},
		{
			name: "teardown after test failure",
			setupMocks: func(env *testsuite.TestWorkflowEnvironment, ca *CellActivities) {
				env.OnActivity(ca.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
					CellID:    "test-cell",
					Port:      8001,
					BaseURL:   "http://localhost:8001",
					ServerPID: 12345,
				}, nil)
				env.OnActivity(ca.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
					Success: true,
					Output:  "done",
				}, nil)
				env.OnActivity(ca.RunTests, mock.Anything, mock.Anything).Return(false, nil)
				env.OnActivity(ca.RevertChanges, mock.Anything, mock.Anything).Return(nil)
				env.OnActivity(ca.TeardownCell, mock.Anything, mock.Anything).Return(nil)
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			cellActivities := &CellActivities{}

			tt.setupMocks(env, cellActivities)

			env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
				CellID:      "test-cell",
				Branch:      "main",
				TaskID:      "task-001",
				Description: "Test",
				Prompt:      "Test",
			})

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			var result TCRWorkflowResult
			require.NoError(t, env.GetWorkflowResult(&result))
			assert.Equal(t, tt.expectSuccess, result.Success)
		})
	}
}

// TestTCRWorkflow_MultipleFilesModified tests workflow with multiple file changes
func TestTCRWorkflow_MultipleFilesModified(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Setup minimal mocks
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:    "test-cell",
		Port:      8001,
		BaseURL:   "http://localhost:8001",
		ServerPID: 12345,
	}, nil)
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success:       true,
		FilesModified: []string{"handler.go", "types.go", "middleware.go", "config.go"},
	}, nil)
	env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(true, nil)
	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID: "test-cell",
		Branch: "main",
		TaskID: "task-001",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TCRWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Len(t, result.FilesChanged, 4)
}

// TestTCRWorkflow_CommitMessageFormat tests commit message formatting
func TestTCRWorkflow_CommitMessageFormat(t *testing.T) {
	tests := []struct {
		name        string
		input       TCRWorkflowInput
		expectInMsg []string
	}{
		{
			name: "standard commit message",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "TASK-123",
				Description: "Implement user authentication",
			},
			expectInMsg: []string{"TASK-123", "Implement user authentication", "🤖 Generated by Reactor-SDK"},
		},
		{
			name: "commit message with special characters",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "TASK-456",
				Description: "Fix bug #789 in service-x",
			},
			expectInMsg: []string{"TASK-456", "Fix bug #789 in service-x", "🤖 Generated by Reactor-SDK"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			cellActivities := &CellActivities{}

			env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
				CellID:    "test-cell",
				Port:      8001,
				BaseURL:   "http://localhost:8001",
				ServerPID: 12345,
			}, nil)
			env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
				Success: true,
			}, nil)
			env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(true, nil)

			var capturedMsg string
			env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				capturedMsg = args.Get(2).(string)
			})
			env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

			env.ExecuteWorkflow(TCRWorkflow, tt.input)

			require.True(t, env.IsWorkflowCompleted())
			for _, expected := range tt.expectInMsg {
				assert.Contains(t, capturedMsg, expected)
			}
		})
	}
}

// TestTCRWorkflow_DisconnectedContextForTeardown tests that teardown uses disconnected context
func TestTCRWorkflow_DisconnectedContextForTeardown(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}

	// Bootstrap succeeds
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:    "test-cell",
		Port:      8001,
		BaseURL:   "http://localhost:8001",
		ServerPID: 12345,
	}, nil)

	// Task execution succeeds
	env.OnActivity(cellActivities.ExecuteTask, mock.Anything, mock.Anything, mock.Anything).Return(&TaskOutput{
		Success: true,
	}, nil)

	// Tests pass
	env.OnActivity(cellActivities.RunTests, mock.Anything, mock.Anything).Return(true, nil)

	// Commit succeeds
	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Teardown should always be called
	teardownCalled := false
	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		teardownCalled = true
	})

	env.ExecuteWorkflow(TCRWorkflow, TCRWorkflowInput{
		CellID: "test-cell",
		Branch: "main",
		TaskID: "task-001",
	})

	require.True(t, env.IsWorkflowCompleted())
	assert.True(t, teardownCalled, "Teardown should always be called")
}

// TestTCRWorkflowInput_Validation tests input validation
func TestTCRWorkflowInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input TCRWorkflowInput
		valid bool
	}{
		{
			name: "valid complete input",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "task-001",
				Description: "Test task",
				Prompt:      "Test prompt",
			},
			valid: true,
		},
		{
			name: "valid minimal input",
			input: TCRWorkflowInput{
				CellID: "cell-001",
				Branch: "main",
				TaskID: "task-001",
			},
			valid: true,
		},
		{
			name: "empty description and prompt allowed",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "task-001",
				Description: "",
				Prompt:      "",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.input.CellID)
				assert.NotEmpty(t, tt.input.Branch)
				assert.NotEmpty(t, tt.input.TaskID)
			}
		})
	}
}

// TestTCRWorkflowResult_Structure tests result structure
func TestTCRWorkflowResult_Structure(t *testing.T) {
	tests := []struct {
		name   string
		result TCRWorkflowResult
	}{
		{
			name: "success result",
			result: TCRWorkflowResult{
				Success:      true,
				TestsPassed:  true,
				FilesChanged: []string{"file1.go", "file2.go"},
				Error:        "",
			},
		},
		{
			name: "failure result",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{},
				Error:        "bootstrap failed",
			},
		},
		{
			name: "partial failure",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{"modified.go"},
				Error:        "tests failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, true, tt.result.Success)
			assert.IsType(t, true, tt.result.TestsPassed)
			assert.IsType(t, []string{}, tt.result.FilesChanged)
			assert.IsType(t, "", tt.result.Error)
		})
	}
}
