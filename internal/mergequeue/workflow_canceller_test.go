package mergequeue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/mocks"
)

// TestWorkflowCanceller_GracefulCancellation tests graceful workflow cancellation.
func TestWorkflowCanceller_GracefulCancellation(t *testing.T) {
	// Create mock Temporal client
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, "test-workflow-1", "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	status, err := canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "test-workflow-1", status.WorkflowID)
	assert.Equal(t, CancellationModeGraceful, status.Mode)
	assert.True(t, status.Success)
	assert.Equal(t, "Graceful cancellation requested", status.Message)
	assert.NotNil(t, status.CompletedAt)
	assert.Contains(t, status.ResourcesFreed, "workflow_execution")
}

// TestWorkflowCanceller_ForceCancellation tests force workflow cancellation.
func TestWorkflowCanceller_ForceCancellation(t *testing.T) {
	// Create mock Temporal client
	mockClient := &mocks.Client{}
	mockClient.On("TerminateWorkflow", MatchAnything{}, "test-workflow-1", "Force cancelled", nil).
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	status, err := canceller.CancelWorkflowForce(ctx, "test-workflow-1")

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "test-workflow-1", status.WorkflowID)
	assert.Equal(t, CancellationModeForce, status.Mode)
	assert.True(t, status.Success)
	assert.Equal(t, "Workflow terminated forcefully", status.Message)
}

// TestWorkflowCanceller_CancellationWithTimeout tests cancellation timeout handling.
func TestWorkflowCanceller_CancellationWithTimeout(t *testing.T) {
	// Create mock Temporal client that simulates delay
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, "test-workflow-1", "").
		Return(context.DeadlineExceeded)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		50*time.Millisecond,
		25*time.Millisecond,
	)

	ctx := context.Background()
	status, _ := canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")

	require.NotNil(t, status)
	assert.False(t, status.Success)
	assert.NotNil(t, status.Error)
	assert.Contains(t, status.Error.Error(), "deadline exceeded")
}

// TestWorkflowCanceller_EmptyWorkflowID tests validation of empty workflow ID.
func TestWorkflowCanceller_EmptyWorkflowID(t *testing.T) {
	mockClient := &mocks.Client{}
	canceller := NewDefaultWorkflowCanceller(mockClient, 500*time.Millisecond, 250*time.Millisecond)

	ctx := context.Background()
	status, err := canceller.CancelWorkflow(ctx, "", CancellationModeGraceful)

	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "workflowID cannot be empty")
}

// TestWorkflowCanceller_CancelMultipleWorkflows tests concurrent cancellation of multiple workflows.
func TestWorkflowCanceller_CancelMultipleWorkflows(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, MatchAnything{}, "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	workflowIDs := []string{"workflow-1", "workflow-2", "workflow-3"}
	results := canceller.CancelWorkflows(ctx, workflowIDs, CancellationModeGraceful)

	assert.Equal(t, len(workflowIDs), len(results))
	for _, id := range workflowIDs {
		assert.NotNil(t, results[id])
		assert.Equal(t, id, results[id].WorkflowID)
		assert.True(t, results[id].Success)
	}
}

// TestWorkflowCanceller_GetCancellationStatus tests retrieval of cached cancellation status.
func TestWorkflowCanceller_GetCancellationStatus(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, "test-workflow-1", "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	cancelStatus, err := canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")
	require.NoError(t, err)

	// Retrieve status from cache
	retrievedStatus := canceller.GetCancellationStatus("test-workflow-1")

	require.NotNil(t, retrievedStatus)
	assert.Equal(t, cancelStatus.WorkflowID, retrievedStatus.WorkflowID)
	assert.Equal(t, cancelStatus.Success, retrievedStatus.Success)
	assert.Equal(t, cancelStatus.Mode, retrievedStatus.Mode)
}

// TestWorkflowCanceller_HasPendingCancellation tests pending cancellation detection.
func TestWorkflowCanceller_HasPendingCancellation(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, "test-workflow-1", "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()

	// Start cancellation in goroutine
	go canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")

	// Check pending status
	time.Sleep(10 * time.Millisecond)
	assert.True(t, canceller.HasPendingCancellation("test-workflow-1"))

	// Wait for completion
	time.Sleep(200 * time.Millisecond)
	assert.False(t, canceller.HasPendingCancellation("test-workflow-1"))
}

// TestWorkflowCanceller_Clear tests clearing of cached statuses.
func TestWorkflowCanceller_Clear(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, MatchAnything{}, "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")
	canceller.CancelWorkflowGraceful(ctx, "test-workflow-2")

	// Verify statuses exist
	assert.NotNil(t, canceller.GetCancellationStatus("test-workflow-1"))
	assert.NotNil(t, canceller.GetCancellationStatus("test-workflow-2"))

	// Clear all
	canceller.Clear()

	// Verify cleared
	assert.Nil(t, canceller.GetCancellationStatus("test-workflow-1"))
	assert.Nil(t, canceller.GetCancellationStatus("test-workflow-2"))
}

// TestNoOpWorkflowCanceller tests no-op canceller for graceful degradation.
func TestNoOpWorkflowCanceller_AllOperations(t *testing.T) {
	canceller := NewNoOpWorkflowCanceller()

	ctx := context.Background()

	// Test graceful cancellation
	status, err := canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.Success)
	assert.Equal(t, CancellationModeGraceful, status.Mode)

	// Test force cancellation
	status, err = canceller.CancelWorkflowForce(ctx, "test-workflow-1")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.Success)
	assert.Equal(t, CancellationModeForce, status.Mode)

	// Test multiple cancellations
	results := canceller.CancelWorkflows(ctx, []string{"w1", "w2", "w3"}, CancellationModeGraceful)
	assert.Equal(t, 3, len(results))

	// Test pending check
	assert.False(t, canceller.HasPendingCancellation("test-workflow-1"))

	// Test get status
	assert.Nil(t, canceller.GetCancellationStatus("test-workflow-1"))

	// Test clear
	canceller.Clear() // Should not panic
}

// TestWorkflowCanceller_ConcurrentCancellations tests thread-safe concurrent cancellations.
func TestWorkflowCanceller_ConcurrentCancellations(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, MatchAnything{}, "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	numGoroutines := 10
	var wg sync.WaitGroup
	results := make(chan *CancellationStatus, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			workflowID := "workflow-" + string(rune(index))
			status, err := canceller.CancelWorkflowGraceful(ctx, workflowID)
			assert.NoError(t, err)
			results <- status
		}(i)
	}

	wg.Wait()
	close(results)

	statusCount := 0
	for status := range results {
		assert.NotNil(t, status)
		assert.True(t, status.Success)
		statusCount++
	}

	assert.Equal(t, numGoroutines, statusCount)
}

// TestWorkflowCanceller_StatusAccuracy tests accuracy of cancellation status reporting.
func TestWorkflowCanceller_StatusAccuracy(t *testing.T) {
	mockClient := &mocks.Client{}
	mockClient.On("CancelWorkflow", MatchAnything{}, "test-workflow-1", "").
		Return(nil)

	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	beforeCancel := time.Now()
	status, err := canceller.CancelWorkflowGraceful(ctx, "test-workflow-1")
	afterCancel := time.Now()

	require.NoError(t, err)
	require.NotNil(t, status)

	// Verify timing
	assert.True(t, status.CancelledAt.After(beforeCancel) || status.CancelledAt.Equal(beforeCancel))
	assert.NotNil(t, status.CompletedAt)
	assert.True(t, status.CompletedAt.After(status.CancelledAt) || status.CompletedAt.Equal(status.CancelledAt))
	assert.True(t, status.CompletedAt.After(beforeCancel) && status.CompletedAt.Before(afterCancel.Add(100*time.Millisecond)))

	// Verify duration calculation
	assert.True(t, status.Duration > 0)
	assert.True(t, status.Duration <= 100*time.Millisecond)
}

// TestWorkflowCanceller_ModeValidation tests that invalid modes are handled properly.
func TestWorkflowCanceller_ModeValidation(t *testing.T) {
	mockClient := &mocks.Client{}
	canceller := NewDefaultWorkflowCanceller(
		mockClient,
		500*time.Millisecond,
		250*time.Millisecond,
	)

	ctx := context.Background()
	status, err := canceller.CancelWorkflow(ctx, "test-workflow-1", "invalid-mode")

	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "unknown cancellation mode")
}

// MatchAnything is a testify matcher that matches any argument
type MatchAnything struct{}

func (m MatchAnything) Matches(x interface{}) bool {
	return true
}

func (m MatchAnything) String() string {
	return "MatchAnything"
}
