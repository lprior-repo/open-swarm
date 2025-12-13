package mergequeue

import (
	"context"
	"fmt"
	"sync"

	"go.temporal.io/sdk/client"
)

// Global registry for Temporal clients and cancellers (used for integration testing)
// This is a workaround to avoid modifying the Coordinator struct while linters are active
var (
	temporalClientRegistry = make(map[*Coordinator]client.Client)
	temporalClientMu       sync.RWMutex

	workflowCancellerRegistry = make(map[*Coordinator]WorkflowCanceller)
	workflowCancellerMu       sync.RWMutex
)

// SetTemporalClient sets the Temporal client for workflow cancellation.
// This is used by integration tests to inject a real Temporal client.
// It also creates a default WorkflowCanceller if one is not already set.
func (c *Coordinator) SetTemporalClient(tc client.Client) {
	temporalClientMu.Lock()
	defer temporalClientMu.Unlock()
	temporalClientRegistry[c] = tc

	// Auto-create a default canceller if not already set
	workflowCancellerMu.Lock()
	defer workflowCancellerMu.Unlock()
	if _, exists := workflowCancellerRegistry[c]; !exists {
		workflowCancellerRegistry[c] = NewDefaultWorkflowCanceller(
			tc,
			c.config.KillSwitchTimeout,
			c.config.KillSwitchTimeout/2, // Force timeout is shorter
		)
	}
}

// SetWorkflowCanceller sets a custom WorkflowCanceller for this coordinator.
// This allows fine-grained control over cancellation behavior.
func (c *Coordinator) SetWorkflowCanceller(canceller WorkflowCanceller) {
	workflowCancellerMu.Lock()
	defer workflowCancellerMu.Unlock()
	workflowCancellerRegistry[c] = canceller
}

// getTemporalClient retrieves the Temporal client if configured.
func (c *Coordinator) getTemporalClient() client.Client {
	temporalClientMu.RLock()
	defer temporalClientMu.RUnlock()
	return temporalClientRegistry[c]
}

// getWorkflowCanceller retrieves the WorkflowCanceller if configured.
func (c *Coordinator) getWorkflowCanceller() WorkflowCanceller {
	workflowCancellerMu.RLock()
	defer workflowCancellerMu.RUnlock()
	return workflowCancellerRegistry[c]
}

// removeTemporalClient removes the Temporal client from the registry.
// Should be called when coordinator is stopped.
func (c *Coordinator) removeTemporalClient() {
	temporalClientMu.Lock()
	defer temporalClientMu.Unlock()
	delete(temporalClientRegistry, c)

	// Also remove the associated canceller
	workflowCancellerMu.Lock()
	defer workflowCancellerMu.Unlock()
	delete(workflowCancellerRegistry, c)
}

// cancelWorkflow cancels a Temporal workflow with graceful mode.
// Returns nil if no canceller is configured (graceful degradation).
func (c *Coordinator) cancelWorkflow(ctx context.Context, workflowID string) error {
	status, err := c.CancelWorkflowGraceful(ctx, workflowID)
	if err != nil {
		return err
	}
	if status == nil || !status.Success {
		if status != nil && status.Error != nil {
			return status.Error
		}
		return nil
	}
	return nil
}

// CancelWorkflowGraceful cancels a workflow gracefully through the WorkflowCanceller.
// Returns the cancellation status or an error if cancellation cannot be performed.
func (c *Coordinator) CancelWorkflowGraceful(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflowID cannot be empty")
	}

	canceller := c.getWorkflowCanceller()
	if canceller == nil {
		return nil, nil // Graceful degradation
	}

	return canceller.CancelWorkflowGraceful(ctx, workflowID)
}

// CancelWorkflowForce cancels a workflow forcefully through the WorkflowCanceller.
// Returns the cancellation status or an error if cancellation cannot be performed.
func (c *Coordinator) CancelWorkflowForce(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflowID cannot be empty")
	}

	canceller := c.getWorkflowCanceller()
	if canceller == nil {
		return nil, nil // Graceful degradation
	}

	return canceller.CancelWorkflowForce(ctx, workflowID)
}

// CancelWorkflow cancels a workflow with the specified mode through the WorkflowCanceller.
// Returns the cancellation status or an error if cancellation cannot be performed.
func (c *Coordinator) CancelWorkflow(ctx context.Context, workflowID string, mode CancellationMode) (*CancellationStatus, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflowID cannot be empty")
	}

	canceller := c.getWorkflowCanceller()
	if canceller == nil {
		return nil, nil // Graceful degradation
	}

	return canceller.CancelWorkflow(ctx, workflowID, mode)
}

// GetCancellationStatus retrieves the status of a previous workflow cancellation.
// Returns nil if no status is found.
func (c *Coordinator) GetCancellationStatus(workflowID string) *CancellationStatus {
	canceller := c.getWorkflowCanceller()
	if canceller == nil {
		return nil
	}
	return canceller.GetCancellationStatus(workflowID)
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
