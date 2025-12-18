package temporal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestEnhancedTCRWorkflow_SuccessfulPath tests complete successful workflow execution
func TestEnhancedTCRWorkflow_SuccessfulPath(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	// Mock activities to return success
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, BootstrapInput{
		CellID: "test-cell",
		Branch: "main",
	}).Return(&BootstrapOutput{
		CellID:     "test-cell",
		Port:       8080,
		WorktreeID: "wt-test",
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return([]string{"file1.go", "file2.go"}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "GenTest",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "LintTest",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "VerifyRED",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "GenImpl",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "VerifyGREEN",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "MultiReview",
		Passed:   true,
	}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "task-1",
		CellID:             "test-cell",
		Branch:             "main",
		Description:        "Test implementation",
		AcceptanceCriteria: "Should work",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     2,
	})

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted(), "workflow should complete")
	require.NoError(t, env.GetWorkflowError(), "workflow should have no error")

	var result *EnhancedTCRResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err, "error getting result")
	require.True(t, result.Success, "result should be successful")
	require.Empty(t, result.Error, "result should have no error")
}

// TestEnhancedTCRWorkflow_BootstrapFailure tests failure during bootstrap
func TestEnhancedTCRWorkflow_BootstrapFailure(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	// Mock bootstrap to fail
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(nil, errors.New("bootstrap failed"))

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:      "task-1",
		CellID:      "test-cell",
		Branch:      "main",
		Description: "Test implementation",
	})

	require.True(t, env.IsWorkflowCompleted(), "workflow should complete")

	var result *EnhancedTCRResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err, "error getting result")
	require.False(t, result.Success, "result should fail")
	require.NotEmpty(t, result.Error, "result should have error")
}

// TestEnhancedTCRWorkflow_GateFailureWithRetry tests gate failure and retry logic
func TestEnhancedTCRWorkflow_GateFailureWithRetry(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	// Mock successful bootstrap and lock acquire
	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "test-cell",
		Port:   8080,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return([]string{"file1.go"}, nil)

	// Mock test generation phase
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "GenTest",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "LintTest",
		Passed:   true,
	}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "VerifyRED",
		Passed:   true,
	}, nil)

	// First regeneration fails on GenImpl
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "GenImpl",
		Passed:   false,
		Error:    "generation failed",
	}, nil)

	env.OnActivity(cellActivities.RevertChanges, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID:             "task-1",
		CellID:             "test-cell",
		Branch:             "main",
		Description:        "Test",
		AcceptanceCriteria: "Should work",
		MaxRetries:         1,
		MaxFixAttempts:     1,
	})

	require.True(t, env.IsWorkflowCompleted(), "workflow should complete")

	var result *EnhancedTCRResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err, "error getting result")
	require.False(t, result.Success, "result should fail")
}

// TestWorkflowLifecycle_SagaCleanup tests that saga pattern ensures cleanup on failure
func TestWorkflowLifecycle_SagaCleanup(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	bootstrapped := false
	locksAcquired := false
	locksReleased := false
	cellTorndown := false

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		bootstrapped = true
	}).Return(&BootstrapOutput{CellID: "test-cell", Port: 8080}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		locksAcquired = true
	}).Return([]string{"file1.go"}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&GateResult{
		GateName: "GenTest",
		Passed:   false,
		Error:    "test generation failed",
	}, nil)

	env.OnActivity(cellActivities.RevertChanges, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		locksReleased = true
	}).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		cellTorndown = true
	}).Return(nil)

	env.ExecuteWorkflow(EnhancedTCRWorkflow, EnhancedTCRInput{
		TaskID: "task-1",
		CellID: "test-cell",
		Branch: "main",
	})

	assert.True(t, bootstrapped, "bootstrap should be called")
	assert.True(t, locksAcquired, "locks should be acquired")
	assert.True(t, locksReleased, "locks should be released (saga pattern)")
	assert.True(t, cellTorndown, "cell should be torn down (saga pattern)")
}
