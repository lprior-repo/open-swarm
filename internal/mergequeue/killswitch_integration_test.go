//go:build integration
// +build integration

package mergequeue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	testTaskQueue       = "merge-queue-integration-test"
	temporalTestAddress = "localhost:7233"
	testTimeout         = 30 * time.Second
)

// TestWorkflow is a simple long-running workflow for testing cancellation
func TestWorkflow(ctx workflow.Context, duration time.Duration) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("TestWorkflow started", "duration", duration)

	// Simulate long-running work with periodic sleep
	timer := workflow.NewTimer(ctx, duration)
	if err := timer.Get(ctx, nil); err != nil {
		return fmt.Errorf("workflow interrupted: %w", err)
	}

	logger.Info("TestWorkflow completed")
	return nil
}

// TestKillSwitch_CancelSingleWorkflow tests that kill switch cancels a single Temporal workflow
func TestKillSwitch_CancelSingleWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalTestAddress,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to connect to Temporal")
	defer c.Close()

	// Setup worker
	w := worker.New(c, testTaskQueue, worker.Options{})
	w.RegisterWorkflow(TestWorkflow)
	go func() {
		_ = w.Run(worker.InterruptCh())
	}()
	defer w.Stop()
	time.Sleep(1 * time.Second)

	// Create coordinator and inject Temporal client
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	coord.SetTemporalClient(c)

	// Create a speculative branch with a running workflow
	workflowID := fmt.Sprintf("kill-test-single-%d", time.Now().UnixNano())
	branchID := "branch-1"

	// Start a long-running workflow (10 seconds)
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: testTaskQueue,
	}
	we, err := c.ExecuteWorkflow(context.Background(), options, TestWorkflow, 10*time.Second)
	require.NoError(t, err, "Failed to start workflow")

	// Add branch to coordinator with workflow ID
	branch := &SpeculativeBranch{
		ID:         branchID,
		Status:     BranchStatusTesting,
		WorkflowID: workflowID,
		Changes: []ChangeRequest{
			{
				ID:            "agent-1",
				FilesModified: []string{"src/test.go"},
			},
		},
	}

	coord.mu.Lock()
	coord.activeBranches[branchID] = branch
	coord.mu.Unlock()

	// Verify workflow is running
	desc, err := c.DescribeWorkflowExecution(ctx, workflowID, "")
	require.NoError(t, err)
	assert.True(t, desc.WorkflowExecutionInfo.Status == 1, "Workflow should be running") // 1 = Running

	// Kill the branch (which should cancel the workflow)
	err = coord.KillFailedBranchWithWorkflow(ctx, branchID, "test failure")
	require.NoError(t, err)

	// Verify branch is marked as killed
	coord.mu.RLock()
	killedBranch := coord.activeBranches[branchID]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.NotNil(t, killedBranch.KilledAt)
	assert.Equal(t, "test failure", killedBranch.KillReason)

	// Wait a bit for cancellation to propagate
	time.Sleep(500 * time.Millisecond)

	// Verify workflow was cancelled
	err = we.Get(ctx, nil)
	assert.Error(t, err, "Workflow should have been cancelled")
}
