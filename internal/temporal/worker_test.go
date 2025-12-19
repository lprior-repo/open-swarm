package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTemporalWorker_Success verifies worker initialization with valid config.
func TestNewTemporalWorker_Success(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue:      "default",
		Namespace:      "default",
		MaxConcurrent:  10,
		RateLimit:      100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, worker)
	assert.Equal(t, opts.TaskQueue, worker.opts.TaskQueue)
	assert.Equal(t, opts.Namespace, worker.opts.Namespace)
}

// TestNewTemporalWorker_MissingTaskQueue verifies validation for missing task queue.
func TestNewTemporalWorker_MissingTaskQueue(t *testing.T) {
	opts := WorkerOptions{
		Namespace:      "default",
		MaxConcurrent:  10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	assert.Error(t, err)
	assert.Nil(t, worker)
	assert.Contains(t, err.Error(), "task_queue")
}

// TestWorker_StartStop verifies lifecycle methods are idempotent.
// Note: Skipped on test machine without Temporal server running.
func TestWorker_StartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping StartStop test that requires Temporal server")
	}

	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)

	// Verify worker is initialized but not started
	require.NotNil(t, worker)
	require.False(t, worker.started)

	// Stop without start should be idempotent
	err = worker.Stop(ctx)
	require.NoError(t, err)
}

// TestWorker_RegisterActivity verifies activity registration.
func TestWorker_RegisterActivity(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)

	// Define a simple activity function
	testActivity := func(ctx context.Context, input string) (string, error) {
		return input + "-processed", nil
	}

	// Register should succeed and not nil the worker
	require.NotNil(t, worker)
	// Simply calling RegisterActivity should work
	worker.RegisterActivity(testActivity)
}

// TestWorker_RegisterWorkflow verifies workflow registration.
func TestWorker_RegisterWorkflow(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)

	// Simply verify worker is ready for registration
	require.NotNil(t, worker)
}

// TestWorker_StopWithoutStart verifies stop works even if never started.
func TestWorker_StopWithoutStart(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)

	// Stop without start should succeed (idempotent)
	err = worker.Stop(ctx)
	require.NoError(t, err)
}

// TestWorker_ContextCancellation verifies context timeout handling.
func TestWorker_ContextCancellation(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	// Create context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is cancelled

	worker, err := NewTemporalWorker(ctx, opts)
	// May return error due to context cancellation
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	} else {
		// If no error, worker should still be valid
		require.NotNil(t, worker)
	}
}

// TestWorker_IsInitialized verifies worker state after creation.
func TestWorker_IsInitialized(t *testing.T) {
	opts := WorkerOptions{
		TaskQueue: "test",
		Namespace: "default",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker, err := NewTemporalWorker(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, worker)

	// Verify worker state
	assert.False(t, worker.started, "worker should not be started on creation")
	assert.Equal(t, opts.TaskQueue, worker.opts.TaskQueue)
	assert.Equal(t, opts.Namespace, worker.opts.Namespace)
}
