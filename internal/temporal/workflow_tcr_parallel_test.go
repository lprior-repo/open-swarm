package temporal

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestParallelTCR_ParallelReviewers tests concurrent reviewer evaluation
func TestParallelTCR_ParallelReviewers(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "parallel-cell-001", Port: 9001,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// Test generation phase
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// Implementation phase
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	// Multiple reviewers execute in parallel
	reviewCount := 0
	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		reviewCount++
	}).Return(&GateResult{
		GateName: "MultiReview",
		Passed:   true,
	}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(ParallelTCRWorkflow, EnhancedTCRInput{
		TaskID:             "parallel-task-001",
		CellID:             "parallel-cell-001",
		Branch:             "main",
		Description:        "Parallel workflow test",
		AcceptanceCriteria: "All gates pass",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
	require.Equal(t, 3, reviewCount, "should have called ExecuteMultiReview 3 times (parallel)")
}

// TestParallelTCR_ParallelFixes tests concurrent fix attempts
func TestParallelTCR_ParallelFixes(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "parallel-cell-002", Port: 9002,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// Test generation phase
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	// Implementation phase
	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	// VerifyGREEN fails first, then passes after parallel fix attempts
	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: false, Error: "tests failed"}, nil).Once()

	// Three parallel fix attempts
	fixCount := 0
	env.OnActivity(enhancedActivities.ExecuteFixFromFeedback, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fixCount++
	}).Return(&GateResult{GateName: "FixFromFeedback", Passed: true}, nil)

	// Second verification passes
	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(ParallelTCRWorkflow, EnhancedTCRInput{
		TaskID:             "parallel-task-002",
		CellID:             "parallel-cell-002",
		Branch:             "main",
		Description:        "Parallel fixes test",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     2,
	})

	require.True(t, env.IsWorkflowCompleted())

	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
	require.Equal(t, 3, fixCount, "should have attempted 3 parallel fixes")
}

// TestParallelTCR_HappyPath tests complete successful parallel execution
func TestParallelTCR_HappyPath(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	cellActivities := &CellActivities{}
	enhancedActivities := &EnhancedActivities{}

	env.OnActivity(cellActivities.BootstrapCell, mock.Anything, mock.Anything).Return(&BootstrapOutput{
		CellID: "parallel-cell-003", Port: 9003,
	}, nil)

	env.OnActivity(enhancedActivities.AcquireFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(
		[]string{"file.go"}, nil)

	// All gates pass
	env.OnActivity(enhancedActivities.ExecuteGenTest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteLintTest, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "LintTest", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyRED, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyRED", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteGenImpl, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "GenImpl", Passed: true}, nil)

	env.OnActivity(enhancedActivities.ExecuteVerifyGREEN, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "VerifyGREEN", Passed: true}, nil)

	// Multiple reviewers in parallel
	env.OnActivity(enhancedActivities.ExecuteMultiReview, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&GateResult{GateName: "MultiReview", Passed: true}, nil)

	env.OnActivity(cellActivities.CommitChanges, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(enhancedActivities.ReleaseFileLocks, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.OnActivity(cellActivities.TeardownCell, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(ParallelTCRWorkflow, EnhancedTCRInput{
		TaskID:             "parallel-task-003",
		CellID:             "parallel-cell-003",
		Branch:             "main",
		Description:        "Parallel happy path",
		MaxRetries:         2,
		MaxFixAttempts:     5,
		ReviewersCount:     3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *EnhancedTCRResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Success)
}
