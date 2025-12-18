package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// ============================================================================
// E2E Integration Tests for Enhanced TCR Workflow
// ============================================================================

// TestEnhancedTCR_HappyPath tests complete successful execution: all gates pass
func TestEnhancedTCR_HappyPath(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	// Mock all activities to succeed
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID:       "e2e-cell-001",
		Port:         8001,
		WorktreeID:   "wt-e2e-001",
		WorktreePath: "/tmp/wt-e2e-001",
		BaseURL:      "http://localhost:8001",
		ServerPID:    12345,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"main.go", "handler.go", "test.go"}, nil)

	// Test Generation Phase - All pass
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// Implementation & Review Phase - All pass on first try
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "e2e-task-001",
		CellID:             "e2e-cell-001",
		Branch:             "main",
		Description:        "E2E test: happy path implementation",
		AcceptanceCriteria: "All gates pass on first attempt",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
	require.Empty(t, result.Error)
	require.Equal(t, 6, len(result.GateResults), "should have 6 gate results")
}

// TestEnhancedTCR_Gate2LintRetry tests Gate 2 (LintTest) failure and retry
func TestEnhancedTCR_Gate2LintRetry(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "e2e-cell-002", Port: 8002,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// GenTest passes
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	// LintTest fails, then passes on retry
	lint1Pass := false
	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		lint1Pass = true
	}).Return(&GateResult{
		GateName: "LintTest",
		Passed:   false,
		Error:    "lint errors found",
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	// Retry: GenTest passes again, LintTest passes
	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.RevertChanges, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "e2e-task-002",
		CellID:             "e2e-cell-002",
		Branch:             "main",
		Description:        "E2E test: Gate 2 lint retry",
		AcceptanceCriteria: "Gate 2 fails then passes after regeneration",
		MaxRetries:         2,
		MaxFixAttempts:     5,
	})

	require.True(t, env.IsWorkflowCompleted())
	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
	require.True(t, lint1Pass, "LintTest should have been called")
}

// TestEnhancedTCR_Gate5TestRetry tests Gate 5 (VerifyGREEN) failure and targeted fix
func TestEnhancedTCR_Gate5TestRetry(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "e2e-cell-003", Port: 8003,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// All test generation gates pass
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// GenImpl passes
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	// VerifyGREEN fails, then passes after targeted fix
	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: false, Error: "tests failed"}, nil).Once()

	env.OnActivity(enhancedActivities.ExecuteFixFromFeedback, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "FixFromFeedback", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "e2e-task-003",
		CellID:             "e2e-cell-003",
		Branch:             "main",
		Description:        "E2E test: Gate 5 test retry with targeted fix",
		AcceptanceCriteria: "VerifyGREEN fails then passes after fix",
		MaxRetries:         2,
		MaxFixAttempts:     5,
	})

	require.True(t, env.IsWorkflowCompleted())
	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
}

// TestEnhancedTCR_MaxRetriesExceeded tests workflow exhaustion when max retries exceeded
func TestEnhancedTCR_MaxRetriesExceeded(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "e2e-cell-004", Port: 8004,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// Test generation passes
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// VerifyGREEN always fails - exhausts all retries
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: false, Error: "all tests failed"}, nil)

	env.OnActivity(enhancedActivities.ExecuteFixFromFeedback, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "FixFromFeedback", Passed: false, Error: "fix failed"}, nil)

	env.OnActivity(cellActivities.RevertChanges, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "e2e-task-004",
		CellID:             "e2e-cell-004",
		Branch:             "main",
		Description:        "E2E test: Max retries exceeded",
		AcceptanceCriteria: "Should fail after exhausting retries",
		MaxRetries:         1,       // Only 1 regeneration attempt
		MaxFixAttempts:     2,       // Only 2 fix attempts
	})

	require.True(t, env.IsWorkflowCompleted())
	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.False(t, result.Success, "should fail when max retries exceeded")
	require.NotEmpty(t, result.Error, "should have error message")
	assert.Contains(t, result.Error, "VerifyGREEN failed", "error should mention VerifyGREEN failure")
}

// TestEnhancedTCR_CompleteScenario tests realistic end-to-end workflow with multiple retries
func TestEnhancedTCR_CompleteScenario(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "e2e-cell-005", Port: 8005,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"api.go", "handler.go", "test.go"}, nil)

	// Test generation phase
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// Implementation phase with some retries
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	// Review phase with feedback-fix cycle
	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil).Once()

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "e2e-task-005",
		CellID:             "e2e-cell-005",
		Branch:             "feature/api-enhancement",
		Description:        "Add new REST API endpoint with full test coverage",
		AcceptanceCriteria: "API returns correct response, all tests pass, reviewers approve",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     3,
	})

	require.True(t, env.IsWorkflowCompleted())
	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
	assert.Equal(t, 6, len(result.GateResults), "should have all 6 gates")
}
