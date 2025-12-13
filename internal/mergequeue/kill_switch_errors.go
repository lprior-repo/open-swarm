// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"fmt"
)

// KillSwitchError represents an error during kill switch operations
type KillSwitchError struct {
	// Operation that failed (e.g., "kill_branch", "cleanup_workflow", "notify_agents")
	Operation string
	// BranchID being killed
	BranchID string
	// Underlying error
	Err error
	// Retry count if applicable
	RetryCount int
	// Whether the operation is recoverable with retry
	Recoverable bool
	// Whether the branch was marked as killed despite the error (graceful degradation)
	BranchMarkedKilled bool
	// Context about what failed
	Context string
}

// Error implements the error interface
func (e *KillSwitchError) Error() string {
	msg := fmt.Sprintf("kill switch error [%s]: branch=%s", e.Operation, e.BranchID)

	if e.Context != "" {
		msg += fmt.Sprintf(" (%s)", e.Context)
	}

	if e.RetryCount > 0 {
		msg += fmt.Sprintf(" (attempt %d)", e.RetryCount)
	}

	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}

	if e.BranchMarkedKilled {
		msg += " [branch marked as killed despite error]"
	}

	return msg
}

// Unwrap returns the underlying error
func (e *KillSwitchError) Unwrap() error {
	return e.Err
}

// CleanupError represents a failure during resource cleanup
type CleanupError struct {
	// Type of resource (e.g., "workflow", "container", "worktree", "notification")
	ResourceType string
	// Identifier of the resource
	ResourceID string
	// What operation failed (e.g., "cancel", "stop", "remove", "send")
	Operation string
	// Underlying error
	Err error
	// Whether cleanup can be retried
	Retryable bool
	// Whether the operation can gracefully degrade
	CanDegrade bool
}

// Error implements the error interface
func (e *CleanupError) Error() string {
	msg := fmt.Sprintf("cleanup error [%s/%s]: failed to %s", e.ResourceType, e.ResourceID, e.Operation)

	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}

	if e.Retryable {
		msg += " (retryable)"
	}

	if e.CanDegrade {
		msg += " (can degrade gracefully)"
	}

	return msg
}

// Unwrap returns the underlying error
func (e *CleanupError) Unwrap() error {
	return e.Err
}

// TimeoutError represents a kill operation that timed out
type TimeoutError struct {
	// Which step timed out
	Step string
	// Branch being killed
	BranchID string
	// Configured timeout
	ConfiguredTimeout int64 // in milliseconds
	// Whether partial progress was made
	PartialProgress bool
	// What was completed before timeout
	CompletedSteps []string
	// What was pending
	PendingSteps []string
}

// Error implements the error interface
func (e *TimeoutError) Error() string {
	msg := fmt.Sprintf("kill operation timed out [%s]: branch=%s, timeout=%dms", e.Step, e.BranchID, e.ConfiguredTimeout)

	if e.PartialProgress {
		msg += fmt.Sprintf(" (partial progress: %d completed, %d pending)", len(e.CompletedSteps), len(e.PendingSteps))
	}

	return msg
}

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	// Maximum number of retry attempts (0 means no retries)
	MaxRetries int
	// Initial delay between retries in milliseconds
	InitialDelayMs int
	// Maximum delay between retries in milliseconds
	MaxDelayMs int
	// Backoff multiplier (default 2.0 for exponential backoff)
	BackoffMultiplier float64
	// Jitter percentage to add randomness (0-100)
	JitterPercent int
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialDelayMs:    100,
		MaxDelayMs:        5000,
		BackoffMultiplier: 2.0,
		JitterPercent:     10,
	}
}

// UserFacingError formats an error for user consumption
type UserFacingError struct {
	// Brief title of the issue
	Title string
	// Detailed user-friendly explanation
	Message string
	// Suggested actions to resolve
	SuggestedActions []string
	// Technical details for debugging
	TechnicalDetails string
	// Branch ID for reference
	BranchID string
}

// Error implements the error interface
func (e *UserFacingError) Error() string {
	msg := fmt.Sprintf("%s: %s (branch=%s)", e.Title, e.Message, e.BranchID)

	if len(e.SuggestedActions) > 0 {
		msg += "\n\nSuggested actions:"
		for i, action := range e.SuggestedActions {
			msg += fmt.Sprintf("\n  %d. %s", i+1, action)
		}
	}

	if e.TechnicalDetails != "" {
		msg += fmt.Sprintf("\n\nTechnical details: %s", e.TechnicalDetails)
	}

	return msg
}
