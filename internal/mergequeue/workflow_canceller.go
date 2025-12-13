// Package mergequeue provides temporal workflow cancellation interface.
package mergequeue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
)

// CancellationMode defines the type of workflow cancellation to perform.
type CancellationMode string

const (
	// CancellationModeGraceful attempts to cancel the workflow gracefully,
	// allowing it to clean up resources and complete pending activities.
	CancellationModeGraceful CancellationMode = "graceful"

	// CancellationModeForce immediately terminates the workflow execution,
	// bypassing graceful shutdown mechanisms.
	CancellationModeForce CancellationMode = "force"
)

// CancellationStatus represents the result of a workflow cancellation attempt.
type CancellationStatus struct {
	// WorkflowID is the ID of the cancelled workflow
	WorkflowID string

	// Success indicates whether the cancellation completed successfully
	Success bool

	// Mode indicates which cancellation mode was used
	Mode CancellationMode

	// Duration is the time taken to complete the cancellation
	Duration time.Duration

	// Error contains any error that occurred during cancellation
	Error error

	// Message provides additional context about the cancellation result
	Message string

	// CancelledAt is the timestamp when the cancellation was requested
	CancelledAt time.Time

	// CompletedAt is the timestamp when the cancellation completed
	CompletedAt *time.Time

	// ResourcesFreed tracks which resources were freed during cancellation
	ResourcesFreed []string
}

// WorkflowCanceller provides a clean interface for cancelling Temporal workflows
// with support for graceful and force cancellation modes, cleanup handling,
// and comprehensive status reporting.
type WorkflowCanceller interface {
	// CancelWorkflow cancels a workflow with the specified mode.
	// It returns a CancellationStatus with detailed information about the cancellation.
	// The context should include cancellation timeout appropriate for the mode.
	CancelWorkflow(ctx context.Context, workflowID string, mode CancellationMode) (*CancellationStatus, error)

	// CancelWorkflowGraceful is a convenience method for graceful cancellation.
	// It's equivalent to CancelWorkflow(ctx, workflowID, CancellationModeGraceful).
	CancelWorkflowGraceful(ctx context.Context, workflowID string) (*CancellationStatus, error)

	// CancelWorkflowForce is a convenience method for force cancellation.
	// It's equivalent to CancelWorkflow(ctx, workflowID, CancellationModeForce).
	CancelWorkflowForce(ctx context.Context, workflowID string) (*CancellationStatus, error)

	// CancelWorkflows cancels multiple workflows concurrently.
	// Returns a map of workflowID to CancellationStatus for all requested cancellations.
	CancelWorkflows(ctx context.Context, workflowIDs []string, mode CancellationMode) map[string]*CancellationStatus

	// GetCancellationStatus retrieves the status of a previous cancellation attempt.
	// Returns nil if the cancellation is not found or has expired.
	GetCancellationStatus(workflowID string) *CancellationStatus

	// HasPendingCancellation checks if a cancellation is still in progress for the workflow.
	HasPendingCancellation(workflowID string) bool

	// Clear removes all stored cancellation statuses (typically called during shutdown).
	Clear()
}

// DefaultWorkflowCanceller implements WorkflowCanceller with standard Temporal client.
type DefaultWorkflowCanceller struct {
	client client.Client

	// Timeout settings for different cancellation modes
	gracefulTimeout time.Duration
	forceTimeout    time.Duration

	// In-memory cache of recent cancellation statuses
	statusCache map[string]*CancellationStatus
	cacheMu     sync.RWMutex

	// Track pending cancellations to prevent duplicate requests
	pendingCancellations map[string]context.CancelFunc
	pendingMu            sync.RWMutex
}

// NewDefaultWorkflowCanceller creates a new WorkflowCanceller with a Temporal client.
func NewDefaultWorkflowCanceller(tc client.Client, gracefulTimeout, forceTimeout time.Duration) *DefaultWorkflowCanceller {
	return &DefaultWorkflowCanceller{
		client:                   tc,
		gracefulTimeout:          gracefulTimeout,
		forceTimeout:             forceTimeout,
		statusCache:              make(map[string]*CancellationStatus),
		pendingCancellations:     make(map[string]context.CancelFunc),
	}
}

// CancelWorkflow cancels a workflow with the specified mode.
func (dwc *DefaultWorkflowCanceller) CancelWorkflow(ctx context.Context, workflowID string, mode CancellationMode) (*CancellationStatus, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflowID cannot be empty")
	}

	if mode == "" {
		mode = CancellationModeGraceful
	}

	// Select timeout based on mode
	timeout := dwc.gracefulTimeout
	if mode == CancellationModeForce {
		timeout = dwc.forceTimeout
	}

	// Create timeout context for cancellation
	cancelCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Track pending cancellation
	dwc.pendingMu.Lock()
	dwc.pendingCancellations[workflowID] = cancel
	dwc.pendingMu.Unlock()

	// Clean up pending cancellation when done
	defer func() {
		dwc.pendingMu.Lock()
		delete(dwc.pendingCancellations, workflowID)
		dwc.pendingMu.Unlock()
	}()

	startTime := time.Now()
	status := &CancellationStatus{
		WorkflowID:  workflowID,
		Mode:        mode,
		CancelledAt: startTime,
		ResourcesFreed: []string{},
	}

	// Perform cancellation
	var err error
	switch mode {
	case CancellationModeGraceful:
		err = dwc.client.CancelWorkflow(cancelCtx, workflowID, "")
		status.Message = "Graceful cancellation requested"

	case CancellationModeForce:
		err = dwc.client.TerminateWorkflow(cancelCtx, workflowID, "Force cancelled", "")
		status.Message = "Workflow terminated forcefully"

	default:
		return nil, fmt.Errorf("unknown cancellation mode: %s", mode)
	}

	// Record completion time and success status
	now := time.Now()
	status.CompletedAt = &now
	status.Duration = now.Sub(startTime)

	if err != nil {
		status.Success = false
		status.Error = err
		status.Message = fmt.Sprintf("%s: %v", status.Message, err)
	} else {
		status.Success = true
		status.ResourcesFreed = append(status.ResourcesFreed, "workflow_execution")
	}

	// Cache the status
	dwc.cacheMu.Lock()
	dwc.statusCache[workflowID] = status
	dwc.cacheMu.Unlock()

	return status, nil
}

// CancelWorkflowGraceful cancels a workflow gracefully.
func (dwc *DefaultWorkflowCanceller) CancelWorkflowGraceful(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	return dwc.CancelWorkflow(ctx, workflowID, CancellationModeGraceful)
}

// CancelWorkflowForce cancels a workflow forcefully.
func (dwc *DefaultWorkflowCanceller) CancelWorkflowForce(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	return dwc.CancelWorkflow(ctx, workflowID, CancellationModeForce)
}

// CancelWorkflows cancels multiple workflows concurrently.
func (dwc *DefaultWorkflowCanceller) CancelWorkflows(ctx context.Context, workflowIDs []string, mode CancellationMode) map[string]*CancellationStatus {
	results := make(map[string]*CancellationStatus, len(workflowIDs))
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	// Create semaphore to limit concurrent cancellations
	sem := make(chan struct{}, 10) // Max 10 concurrent cancellations

	for _, workflowID := range workflowIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			sem <- struct{}{} // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			status, err := dwc.CancelWorkflow(ctx, id, mode)
			if err != nil && status == nil {
				status = &CancellationStatus{
					WorkflowID:  id,
					Mode:        mode,
					Success:     false,
					Error:       err,
					CancelledAt: time.Now(),
					Message:     fmt.Sprintf("cancellation failed: %v", err),
				}
			}

			resultsMu.Lock()
			results[id] = status
			resultsMu.Unlock()
		}(workflowID)
	}

	wg.Wait()
	return results
}

// GetCancellationStatus retrieves the status of a previous cancellation attempt.
func (dwc *DefaultWorkflowCanceller) GetCancellationStatus(workflowID string) *CancellationStatus {
	dwc.cacheMu.RLock()
	defer dwc.cacheMu.RUnlock()
	return dwc.statusCache[workflowID]
}

// HasPendingCancellation checks if a cancellation is still in progress.
func (dwc *DefaultWorkflowCanceller) HasPendingCancellation(workflowID string) bool {
	dwc.pendingMu.RLock()
	defer dwc.pendingMu.RUnlock()
	_, exists := dwc.pendingCancellations[workflowID]
	return exists
}

// Clear removes all stored cancellation statuses.
func (dwc *DefaultWorkflowCanceller) Clear() {
	dwc.cacheMu.Lock()
	dwc.statusCache = make(map[string]*CancellationStatus)
	dwc.cacheMu.Unlock()

	dwc.pendingMu.Lock()
	// Cancel all pending cancellations
	for _, cancel := range dwc.pendingCancellations {
		cancel()
	}
	dwc.pendingCancellations = make(map[string]context.CancelFunc)
	dwc.pendingMu.Unlock()
}

// NoOpWorkflowCanceller is a no-op implementation for testing and graceful degradation.
type NoOpWorkflowCanceller struct{}

// NewNoOpWorkflowCanceller creates a no-op canceller.
func NewNoOpWorkflowCanceller() *NoOpWorkflowCanceller {
	return &NoOpWorkflowCanceller{}
}

func (nwc *NoOpWorkflowCanceller) CancelWorkflow(ctx context.Context, workflowID string, mode CancellationMode) (*CancellationStatus, error) {
	return &CancellationStatus{
		WorkflowID:  workflowID,
		Mode:        mode,
		Success:     true,
		CancelledAt: time.Now(),
		Message:     "no-op canceller (no action taken)",
	}, nil
}

func (nwc *NoOpWorkflowCanceller) CancelWorkflowGraceful(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	return nwc.CancelWorkflow(ctx, workflowID, CancellationModeGraceful)
}

func (nwc *NoOpWorkflowCanceller) CancelWorkflowForce(ctx context.Context, workflowID string) (*CancellationStatus, error) {
	return nwc.CancelWorkflow(ctx, workflowID, CancellationModeForce)
}

func (nwc *NoOpWorkflowCanceller) CancelWorkflows(ctx context.Context, workflowIDs []string, mode CancellationMode) map[string]*CancellationStatus {
	results := make(map[string]*CancellationStatus, len(workflowIDs))
	for _, id := range workflowIDs {
		results[id] = &CancellationStatus{
			WorkflowID:  id,
			Mode:        mode,
			Success:     true,
			CancelledAt: time.Now(),
			Message:     "no-op canceller (no action taken)",
		}
	}
	return results
}

func (nwc *NoOpWorkflowCanceller) GetCancellationStatus(workflowID string) *CancellationStatus {
	return nil
}

func (nwc *NoOpWorkflowCanceller) HasPendingCancellation(workflowID string) bool {
	return false
}

func (nwc *NoOpWorkflowCanceller) Clear() {}
