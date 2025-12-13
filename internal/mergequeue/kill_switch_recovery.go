// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// KillOperationState tracks the state of a kill operation for recovery
type KillOperationState struct {
	// Branch being killed
	BranchID string
	// Current step in the kill operation
	CurrentStep string
	// Completed steps for rollback
	CompletedSteps []string
	// Start time of the operation
	StartTime time.Time
	// Last error encountered
	LastError error
	// Number of retries attempted
	RetryAttempt int
	// Whether the branch was marked as killed
	BranchMarkedKilled bool
	// Original kill reason
	KillReason string
}

// RetryStrategy encapsulates retry logic for kill operations
type RetryStrategy struct {
	config RetryConfig
	state  KillOperationState
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(branchID, reason string, config RetryConfig) *RetryStrategy {
	return &RetryStrategy{
		config: config,
		state: KillOperationState{
			BranchID:      branchID,
			StartTime:     time.Now(),
			CompletedSteps: []string{},
			KillReason:    reason,
		},
	}
}

// ShouldRetry determines if an operation should be retried
func (rs *RetryStrategy) ShouldRetry() bool {
	if rs.config.MaxRetries <= 0 {
		return false
	}
	return rs.state.RetryAttempt < rs.config.MaxRetries
}

// NextRetryDelay calculates the delay before the next retry
func (rs *RetryStrategy) NextRetryDelay() time.Duration {
	if !rs.ShouldRetry() {
		return 0
	}

	// Calculate exponential backoff
	delayMs := float64(rs.config.InitialDelayMs) * math.Pow(rs.config.BackoffMultiplier, float64(rs.state.RetryAttempt))

	// Cap at max delay
	if delayMs > float64(rs.config.MaxDelayMs) {
		delayMs = float64(rs.config.MaxDelayMs)
	}

	// Add jitter
	if rs.config.JitterPercent > 0 {
		jitterRange := delayMs * float64(rs.config.JitterPercent) / 100.0
		jitter := rand.Float64() * jitterRange
		delayMs += jitter
	}

	return time.Duration(delayMs) * time.Millisecond
}

// RecordStep marks a step as completed for recovery tracking
func (rs *RetryStrategy) RecordStep(stepName string) {
	rs.state.CurrentStep = stepName
	rs.state.CompletedSteps = append(rs.state.CompletedSteps, stepName)
}

// RecordError records an error and increments retry counter
func (rs *RetryStrategy) RecordError(err error) {
	rs.state.LastError = err
	rs.state.RetryAttempt++
}

// GetState returns the current operation state
func (rs *RetryStrategy) GetState() KillOperationState {
	return rs.state
}

// ExecuteWithRetry runs an operation with automatic retry logic
func ExecuteWithRetry(ctx context.Context, branchID string, op func() error, config RetryConfig) error {
	strategy := NewRetryStrategy(branchID, "operation", config)

	for {
		// Check if we should proceed with this attempt
		if strategy.state.RetryAttempt > 0 {
			delay := strategy.NextRetryDelay()
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("retry loop cancelled: %w", ctx.Err())
			}
		}

		// Execute the operation
		err := op()
		if err == nil {
			return nil
		}

		// Record the error
		strategy.RecordError(err)

		// Check if we should retry
		if !strategy.ShouldRetry() {
			return &KillSwitchError{
				Operation:   "execute_with_retry",
				BranchID:    branchID,
				Err:         err,
				RetryCount:  strategy.state.RetryAttempt,
				Recoverable: false,
				Context:     fmt.Sprintf("max retries (%d) exceeded", config.MaxRetries),
			}
		}
	}
}

// RollbackState represents what can be rolled back
type RollbackState struct {
	// What has been changed
	Changes []RollbackAction
	// Whether rollback was successful
	Success bool
	// Errors encountered during rollback
	Errors []error
}

// RollbackAction represents a single rollback action
type RollbackAction struct {
	// Description of what to rollback
	Description string
	// Function to rollback
	Action func(ctx context.Context) error
	// Whether this action was already executed
	Executed bool
	// Error if rollback failed
	Error error
}

// KillSwitchRecoveryManager manages recovery and rollback for kill operations
type KillSwitchRecoveryManager struct {
	branchID      string
	rollbackStack []*RollbackAction
	operationLog  []string
}

// NewKillSwitchRecoveryManager creates a new recovery manager
func NewKillSwitchRecoveryManager(branchID string) *KillSwitchRecoveryManager {
	return &KillSwitchRecoveryManager{
		branchID:      branchID,
		rollbackStack: make([]*RollbackAction, 0),
		operationLog:  make([]string, 0),
	}
}

// RegisterRollback registers a rollback action in LIFO order (stack)
func (m *KillSwitchRecoveryManager) RegisterRollback(description string, action func(ctx context.Context) error) {
	m.rollbackStack = append(m.rollbackStack, &RollbackAction{
		Description: description,
		Action:      action,
		Executed:    false,
	})
}

// LogOperation logs an operation for debugging
func (m *KillSwitchRecoveryManager) LogOperation(operation string) {
	m.operationLog = append(m.operationLog, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339Nano), operation))
}

// Rollback executes all registered rollback actions in reverse order
func (m *KillSwitchRecoveryManager) Rollback(ctx context.Context) *RollbackState {
	state := &RollbackState{
		Changes: make([]RollbackAction, len(m.rollbackStack)),
		Success: true,
		Errors:  make([]error, 0),
	}

	// Execute rollback actions in reverse order (LIFO)
	for i := len(m.rollbackStack) - 1; i >= 0; i-- {
		action := m.rollbackStack[i]

		// Execute the rollback action
		if err := action.Action(ctx); err != nil {
			action.Error = err
			state.Success = false
			state.Errors = append(state.Errors, fmt.Errorf("rollback %s failed: %w", action.Description, err))
		} else {
			action.Executed = true
		}

		// Copy to state for return
		state.Changes[i] = *action
	}

	return state
}

// GetOperationLog returns the operation log for debugging
func (m *KillSwitchRecoveryManager) GetOperationLog() []string {
	return m.operationLog
}

// BranchStateSnapshot captures the state of a branch for rollback
type BranchStateSnapshot struct {
	BranchID   string
	Status     BranchStatus
	KilledAt   *time.Time
	KillReason string
}

// SaveBranchState creates a snapshot of branch state before modification
func (m *KillSwitchRecoveryManager) SaveBranchState(branch *SpeculativeBranch) {
	killedAtCopy := (*time.Time)(nil)
	if branch.KilledAt != nil {
		copy := *branch.KilledAt
		killedAtCopy = &copy
	}

	snapshot := BranchStateSnapshot{
		BranchID:   branch.ID,
		Status:     branch.Status,
		KilledAt:   killedAtCopy,
		KillReason: branch.KillReason,
	}

	m.LogOperation(fmt.Sprintf("Branch state snapshot: status=%s, killedAt=%v, reason=%q", snapshot.Status, snapshot.KilledAt, snapshot.KillReason))
}

// IsCascadingKill checks if a kill operation is cascading to dependent branches
type CascadeKillValidator struct {
	// Maximum recursion depth allowed
	MaxDepth int
	// Current recursion depth
	CurrentDepth int
	// Branches already being killed
	ProcessingBranches map[string]bool
	// Estimated branches to be killed
	EstimatedBranchCount int
	// Maximum branches that can be killed in one cascade
	MaxBranchesPerCascade int
}

// NewCascadeKillValidator creates a new cascade validator
func NewCascadeKillValidator(maxDepth, maxBranches int) *CascadeKillValidator {
	return &CascadeKillValidator{
		MaxDepth:             maxDepth,
		ProcessingBranches:   make(map[string]bool),
		MaxBranchesPerCascade: maxBranches,
	}
}

// CanCascade checks if cascading is safe
func (cv *CascadeKillValidator) CanCascade(branchID string) error {
	// Check depth
	if cv.CurrentDepth >= cv.MaxDepth {
		return &KillSwitchError{
			Operation:   "cascade_kill",
			BranchID:    branchID,
			Err:         fmt.Errorf("max cascade depth (%d) exceeded", cv.MaxDepth),
			Recoverable: false,
			Context:     "cascade depth limit reached",
		}
	}

	// Check for circular references
	if cv.ProcessingBranches[branchID] {
		return &KillSwitchError{
			Operation:   "cascade_kill",
			BranchID:    branchID,
			Err:         fmt.Errorf("circular cascade detected"),
			Recoverable: false,
			Context:     fmt.Sprintf("branch %s is already being processed", branchID),
		}
	}

	// Check total branches
	if cv.EstimatedBranchCount >= cv.MaxBranchesPerCascade {
		return &KillSwitchError{
			Operation:   "cascade_kill",
			BranchID:    branchID,
			Err:         fmt.Errorf("cascade would exceed max branches limit"),
			Recoverable: false,
			Context:     fmt.Sprintf("estimated %d branches would be killed, max is %d", cv.EstimatedBranchCount, cv.MaxBranchesPerCascade),
		}
	}

	return nil
}

// EnterBranch marks a branch as being processed
func (cv *CascadeKillValidator) EnterBranch(branchID string) {
	cv.ProcessingBranches[branchID] = true
	cv.CurrentDepth++
	cv.EstimatedBranchCount++
}

// ExitBranch marks a branch as no longer being processed
func (cv *CascadeKillValidator) ExitBranch(branchID string) {
	delete(cv.ProcessingBranches, branchID)
	cv.CurrentDepth--
}
