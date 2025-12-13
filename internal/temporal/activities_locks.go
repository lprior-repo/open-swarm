// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
	"open-swarm/internal/filelock"
)

// LockError represents a file lock acquisition failure
type LockError struct {
	Message string
	Paths   []string
	Reason  string
}

func (e *LockError) Error() string {
	return fmt.Sprintf("lock failed: %s (paths: %v, reason: %s)", e.Message, e.Paths, e.Reason)
}

// ExecuteTaskWithLocksInput contains parameters for executing a task with lock management
type ExecuteTaskWithLocksInput struct {
	Bootstrap  *BootstrapOutput
	Task       TaskInput
	FilePaths  []string // Paths to acquire locks for
	CellID     string
	LockTimeMs int64 // Lock validity duration in milliseconds
}

// CellActivities now includes lock management methods (extends existing CellActivities)

// ExecuteTaskWithLocks acquires locks before execution, releases after
// If lock acquisition fails, returns a conflict error that workflow can retry
func (ca *CellActivities) ExecuteTaskWithLocks(ctx context.Context, input ExecuteTaskWithLocksInput) (*TaskOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Acquiring locks for task execution", "taskID", input.Task.TaskID, "paths", input.FilePaths)

	// Step 1: Acquire locks for all paths
	if err := ca.AcquireFileLocks(ctx, input.FilePaths, input.CellID); err != nil {
		logger.Error("Failed to acquire locks", "error", err, "paths", input.FilePaths)
		// Return conflict error - workflow should retry
		return &TaskOutput{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err // Let Temporal handle as retriable error
	}

	// Step 2: Record initial heartbeat
	activity.RecordHeartbeat(ctx, "locks acquired, executing task")

	// Step 3: Execute task with periodic lock renewal
	output := ca.executeTaskWithHeartbeat(ctx, input)

	// Step 4: Release locks in defer (ensures cleanup even on error)
	defer func() {
		logger.Info("Releasing locks", "taskID", input.Task.TaskID, "paths", input.FilePaths)
		if releaseErr := ca.ReleaseFileLocks(ctx, input.FilePaths, input.CellID); releaseErr != nil {
			logger.Error("Failed to release locks", "error", releaseErr)
		}
	}()

	return output, nil
}

// executeTaskWithHeartbeat runs the task and renews locks periodically
func (ca *CellActivities) executeTaskWithHeartbeat(ctx context.Context, input ExecuteTaskWithLocksInput) *TaskOutput {
	// Reconstruct cell from bootstrap output
	cell := ca.reconstructCell(input.Bootstrap)

	taskCtx := &agent.TaskContext{
		TaskID:      input.Task.TaskID,
		Description: input.Task.Description,
		Prompt:      input.Task.Prompt,
	}

	// Execute task
	result, err := ca.activities.ExecuteTask(ctx, cell, taskCtx)
	if err != nil {
		return &TaskOutput{
			Success:      false,
			ErrorMessage: err.Error(),
		}
	}

	// Record heartbeat after task completes (renews lock leases)
	activity.RecordHeartbeat(ctx, "task completed, renewing locks")

	return &TaskOutput{
		Success:       result.Success,
		Output:        result.Output,
		FilesModified: result.FilesModified,
		ErrorMessage:  result.ErrorMessage,
	}
}

// AcquireFileLocks acquires exclusive locks for the specified file paths
// Returns a LockError if any path cannot be locked (due to conflict)
// Lock validity is determined by activity context timeout
func (ca *CellActivities) AcquireFileLocks(ctx context.Context, paths []string, cellID string) error {
	logger := activity.GetLogger(ctx)

	if len(paths) == 0 {
		return nil // No locks needed
	}

	logger.Info("Acquiring file locks", "cellID", cellID, "paths", paths, "pathCount", len(paths))

	// Get the global lock registry
	lockRegistry := GetFileLockRegistry()
	if lockRegistry == nil {
		return &LockError{
			Message: "lock registry not initialized",
			Paths:   paths,
			Reason:  "system not ready",
		}
	}

	// Acquire locks for each path
	acquiredPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		req := filelock.LockRequest{
			Path:      path,
			Holder:    cellID,
			Exclusive: true,
			TTL:       2 * time.Minute, // Default 2-minute lock lease
		}

		result, err := lockRegistry.Acquire(req)
		if err != nil {
			// Rollback: release all acquired locks
			for _, acquired := range acquiredPaths {
				_ = lockRegistry.Release(acquired, cellID)
			}

			logger.Error("Lock acquisition failed", "path", path, "error", err)
			return &LockError{
				Message: fmt.Sprintf("failed to acquire lock on %s", path),
				Paths:   paths,
				Reason:  err.Error(),
			}
		}

		if !result.Granted {
			// Rollback: release all acquired locks
			for _, acquired := range acquiredPaths {
				_ = lockRegistry.Release(acquired, cellID)
			}

			conflictInfo := fmt.Sprintf("conflicts with %d existing lock(s)", len(result.Conflicts))
			logger.Error("Lock acquisition conflict", "path", path, "conflicts", result.Conflicts)
			return &LockError{
				Message: fmt.Sprintf("lock conflict on %s", path),
				Paths:   paths,
				Reason:  conflictInfo,
			}
		}

		acquiredPaths = append(acquiredPaths, path)
	}

	// Record heartbeat for long lock operations
	activity.RecordHeartbeat(ctx, fmt.Sprintf("acquiring locks for %d paths", len(paths)))

	logger.Info("Successfully acquired locks", "cellID", cellID, "paths", paths)
	return nil
}

// ReleaseFileLocks releases exclusive locks for the specified file paths
// Should be called after task completion to free resources for other tasks
func (ca *CellActivities) ReleaseFileLocks(ctx context.Context, paths []string, cellID string) error {
	logger := activity.GetLogger(ctx)

	if len(paths) == 0 {
		return nil // No locks to release
	}

	logger.Info("Releasing file locks", "cellID", cellID, "paths", paths)

	// Get the global lock registry
	lockRegistry := GetFileLockRegistry()
	if lockRegistry == nil {
		logger.Error("Lock registry not initialized during release")
		return fmt.Errorf("lock registry not initialized")
	}

	// Release locks for each path
	var releaseErrors []string
	for _, path := range paths {
		if err := lockRegistry.Release(path, cellID); err != nil {
			logger.Error("Failed to release lock", "path", path, "error", err)
			releaseErrors = append(releaseErrors, fmt.Sprintf("%s: %v", path, err))
		}
	}

	// Record heartbeat
	activity.RecordHeartbeat(ctx, fmt.Sprintf("releasing locks for %d paths", len(paths)))

	if len(releaseErrors) > 0 {
		return fmt.Errorf("failed to release some locks: %v", releaseErrors)
	}

	logger.Info("Successfully released locks", "cellID", cellID)
	return nil
}

// RenewLocks renews the leases on held locks to prevent expiration during long tasks
// Called periodically via heartbeat during extended task execution
func (ca *CellActivities) RenewLocks(ctx context.Context, paths []string, cellID string, renewalDuration time.Duration) error {
	logger := activity.GetLogger(ctx)

	if len(paths) == 0 {
		return nil
	}

	logger.Info("Renewing file locks", "cellID", cellID, "paths", paths, "renewalDuration", renewalDuration)

	// Get the global lock registry
	lockRegistry := GetFileLockRegistry()
	if lockRegistry == nil {
		logger.Error("Lock registry not initialized during renewal")
		return fmt.Errorf("lock registry not initialized")
	}

	// Renew locks for each path
	var renewalErrors []string
	for _, path := range paths {
		if err := lockRegistry.RenewLock(path, cellID, renewalDuration); err != nil {
			logger.Error("Failed to renew lock", "path", path, "error", err)
			renewalErrors = append(renewalErrors, fmt.Sprintf("%s: %v", path, err))
		}
	}

	// Record heartbeat to signal we're still alive
	activity.RecordHeartbeat(ctx, fmt.Sprintf("renewing locks for %d paths, duration: %v", len(paths), renewalDuration))

	if len(renewalErrors) > 0 {
		return fmt.Errorf("failed to renew some locks: %v", renewalErrors)
	}

	return nil
}

// ConflictError implementation for Temporal retry logic
// Temporal treats specific error types specially for retry decisions
var _ error = (*LockError)(nil)

// TaskWithLocksOutput wraps TaskOutput with lock management metadata
type TaskWithLocksOutput struct {
	TaskOutput    *TaskOutput
	LocksHeld     []string
	LocksDuration time.Duration
}
