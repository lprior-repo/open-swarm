package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// WorkerOptions contains configuration for TemporalWorker.
type WorkerOptions struct {
	// TaskQueue is the task queue name for this worker.
	TaskQueue string
	// Namespace is the Temporal namespace (default: "default").
	Namespace string
	// MaxConcurrent is max concurrent activities/workflows (default: 10).
	MaxConcurrent int
	// RateLimit is max activities per second (default: 100).
	RateLimit int
}

// TemporalWorker manages Temporal client and worker lifecycle.
type TemporalWorker struct {
	client   client.Client
	worker   worker.Worker
	opts     WorkerOptions
	started  bool
	mu       sync.RWMutex
}

// NewTemporalWorker creates and initializes a new TemporalWorker.
func NewTemporalWorker(ctx context.Context, opts WorkerOptions) (*TemporalWorker, error) {
	// Validate required options
	if opts.TaskQueue == "" {
		return nil, errors.New("task_queue is required")
	}

	// Set defaults
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 10
	}
	if opts.RateLimit <= 0 {
		opts.RateLimit = 100
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		Namespace: opts.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	// Create worker
	w := worker.New(c, opts.TaskQueue, worker.Options{
		MaxConcurrentActivityTaskPollers: opts.MaxConcurrent,
		MaxConcurrentWorkflowTaskPollers: opts.MaxConcurrent,
	})

	return &TemporalWorker{
		client:  c,
		worker:  w,
		opts:    opts,
		started: false,
	}, nil
}

// Start begins the worker's execution loop.
// Idempotent: calling Start multiple times is safe.
func (w *TemporalWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Idempotent: if already started, do nothing
	if w.started {
		return nil
	}

	if w.worker == nil {
		return errors.New("worker not initialized")
	}

	// Start the worker
	if err := w.worker.Start(); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	w.started = true
	return nil
}

// Stop gracefully shuts down the worker.
// Idempotent: calling Stop multiple times is safe.
func (w *TemporalWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Idempotent: if not started, do nothing
	if !w.started {
		return nil
	}

	// Stop the worker gracefully
	w.worker.Stop()
	w.started = false

	return nil
}

// RegisterActivity registers an activity function with the worker.
func (w *TemporalWorker) RegisterActivity(activity interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.worker != nil {
		w.worker.RegisterActivity(activity)
	}
}

// RegisterWorkflow registers a workflow function with the worker.
func (w *TemporalWorker) RegisterWorkflow(workflow interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.worker != nil {
		w.worker.RegisterWorkflow(workflow)
	}
}

// Close closes the Temporal client connection.
func (w *TemporalWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ensure worker is stopped
	if w.started {
		w.worker.Stop()
		w.started = false
	}

	// Close client
	if w.client != nil {
		w.client.Close()
	}

	return nil
}
