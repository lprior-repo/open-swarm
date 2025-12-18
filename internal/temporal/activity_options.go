// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Shared activity timeout constants
const (
	// DefaultStartToCloseTimeout is the standard timeout for most activities
	DefaultStartToCloseTimeout = 10 * time.Minute

	// DefaultHeartbeatTimeout is the standard heartbeat timeout
	// Increased for LLM activities which can take several minutes
	DefaultHeartbeatTimeout = 2 * time.Minute

	// CleanupStartToCloseTimeout is the timeout for cleanup/teardown operations
	CleanupStartToCloseTimeout = 2 * time.Minute

	// CleanupMaxAttempts is the retry count for cleanup operations
	CleanupMaxAttempts = 3
)

// GetNonIdempotentActivityOptions returns activity options for non-idempotent operations.
// These operations should not be retried automatically (MaximumAttempts = 1).
// Use for: task execution, code generation, agent invocations.
func GetNonIdempotentActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: DefaultStartToCloseTimeout,
		HeartbeatTimeout:    DefaultHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // Don't retry non-idempotent operations
		},
	}
}

// GetCleanupActivityOptions returns activity options for cleanup/teardown operations.
// These operations use retries since cleanup should be resilient.
// Use for: cell teardown, lock release, resource cleanup in saga patterns.
func GetCleanupActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: CleanupStartToCloseTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: CleanupMaxAttempts,
		},
	}
}

// WithNonIdempotentOptions applies non-idempotent activity options to the workflow context.
func WithNonIdempotentOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, GetNonIdempotentActivityOptions())
}

// WithCleanupOptions applies cleanup activity options to the workflow context.
func WithCleanupOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, GetCleanupActivityOptions())
}

// NewSagaContext creates a disconnected context with cleanup options for saga pattern cleanup.
// Use this in defer blocks to ensure cleanup activities run even if the workflow is cancelled.
// Returns the saga context and a cancel function (which can be ignored for cleanup scenarios).
func NewSagaContext(ctx workflow.Context) (workflow.Context, workflow.CancelFunc) {
	disconnCtx, cancel := workflow.NewDisconnectedContext(ctx)
	return WithCleanupOptions(disconnCtx), cancel
}
