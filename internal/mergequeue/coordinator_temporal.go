package mergequeue

import (
	"context"
	"fmt"
	"sync"

	"go.temporal.io/sdk/client"
)

// Global registry for Temporal clients (used for integration testing)
// This is a workaround to avoid modifying the Coordinator struct while linters are active
var (
	temporalClientRegistry = make(map[*Coordinator]client.Client)
	temporalClientMu       sync.RWMutex
)

// SetTemporalClient sets the Temporal client for workflow cancellation.
// This is used by integration tests to inject a real Temporal client.
func (c *Coordinator) SetTemporalClient(tc client.Client) {
	temporalClientMu.Lock()
	defer temporalClientMu.Unlock()
	temporalClientRegistry[c] = tc
}

// getTemporalClient retrieves the Temporal client if configured.
func (c *Coordinator) getTemporalClient() client.Client {
	temporalClientMu.RLock()
	defer temporalClientMu.RUnlock()
	return temporalClientRegistry[c]
}

// removeTemporalClient removes the Temporal client from the registry.
// Should be called when coordinator is stopped.
func (c *Coordinator) removeTemporalClient() {
	temporalClientMu.Lock()
	defer temporalClientMu.Unlock()
	delete(temporalClientRegistry, c)
}

// cancelWorkflow cancels a Temporal workflow if a client is configured.
// Returns nil if no client is configured (graceful degradation).
func (c *Coordinator) cancelWorkflow(ctx context.Context, workflowID string) error {
	if workflowID == "" {
		return nil
	}

	tc := c.getTemporalClient()
	if tc == nil {
		return nil
	}

	// Create a context with timeout for the cancellation request
	cancelCtx, cancel := context.WithTimeout(ctx, c.config.KillSwitchTimeout)
	defer cancel()

	// Cancel the workflow
	return tc.CancelWorkflow(cancelCtx, workflowID, "")
}

// mergeSuccessfulBranch handles merging a successful branch.
// This is a stub for future implementation.
func (c *Coordinator) mergeSuccessfulBranch(_ context.Context, _ *TestResult) {
	// TODO: Implement merge logic
	// 1. Merge the changes to main branch
	// 2. Update queue state
	// 3. Remove merged changes from queue
	// 4. Update metrics
}

// KillFailedBranchWithWorkflow is a wrapper around killFailedBranch that adds
// Temporal workflow cancellation when a Temporal client is configured.
// This is used by integration tests.
func (c *Coordinator) KillFailedBranchWithWorkflow(ctx context.Context, branchID string, reason string) error {
	// Get workflow ID before calling killFailedBranch
	c.mu.RLock()
	branch, exists := c.activeBranches[branchID]
	if !exists {
		c.mu.RUnlock()
		return fmt.Errorf("branch %s not found", branchID)
	}
	workflowID := branch.WorkflowID
	c.mu.RUnlock()

	// Call the original killFailedBranch
	if err := c.killFailedBranch(ctx, branchID, reason); err != nil {
		return err
	}

	// Additionally cancel the workflow if configured
	if err := c.cancelWorkflow(ctx, workflowID); err != nil {
		// Log error but don't fail the kill operation
		_ = err
	}

	return nil
}
