// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package slices

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
)

// ============================================================================
// TIMEOUT CONFIGURATIONS
// ============================================================================

const (
	// AgentExecutionTimeout is the timeout for individual agent operations
	AgentExecutionTimeout = 5 * time.Minute

	// WorkflowTimeout is the timeout for full TCR workflow execution
	WorkflowTimeout = 30 * time.Minute

	// EnhancedTCRStartToCloseTimeout is the maximum time for executing enhanced TCR activities
	EnhancedTCRStartToCloseTimeout = 10 * time.Minute

	// EnhancedTCRHeartbeatTimeout is the heartbeat timeout for enhanced TCR activities
	EnhancedTCRHeartbeatTimeout = 30 * time.Second

	// EnhancedTCRRetryCleanupTimeout is the timeout for cleanup/retry operations
	EnhancedTCRRetryCleanupTimeout = 2 * time.Minute

	// EnhancedTCRRetryMaxAttempts is the maximum number of retry attempts for TCR operations
	EnhancedTCRRetryMaxAttempts = 3
)

// ============================================================================
// RETRY POLICIES
// ============================================================================

// DefaultActivityRetryPolicy returns a standard retry policy with 3 attempts and exponential backoff
// Used for idempotent activities that can safely retry
func DefaultActivityRetryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	}
}

// FileConflictRetryPolicy returns a retry policy optimized for file lock conflicts
// 5 attempts with longer backoff to allow lock holders to release resources
func FileConflictRetryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    60 * time.Second,
		MaximumAttempts:    5,
	}
}

// AgentExecutionTimeoutPolicy returns timeout configuration for agent task execution
// 5 minute timeout suitable for individual agent operations (tests, execution, commits)
func AgentExecutionTimeoutPolicy() time.Duration {
	return AgentExecutionTimeout
}

// WorkflowTimeoutPolicy returns timeout configuration for full TCR workflow execution
// 30 minute timeout suitable for complete bootstrap-execute-test-commit/revert-teardown cycle
func WorkflowTimeoutPolicy() time.Duration {
	return WorkflowTimeout
}

// ============================================================================
// ACTIVITY OPTIONS
// ============================================================================

// ActivityOptions holds retry policy and timeout configuration
type ActivityOptions struct {
	RetryPolicy *temporal.RetryPolicy
	Timeout     time.Duration
	Heartbeat   time.Duration
}

// WithRetryPolicy creates ActivityOptions with specified retry policy
func WithRetryPolicy(policy *temporal.RetryPolicy) ActivityOptions {
	return ActivityOptions{
		RetryPolicy: policy,
		Timeout:     AgentExecutionTimeoutPolicy(),
	}
}

// WithTimeout creates ActivityOptions with specified timeout
func WithTimeout(timeout time.Duration) ActivityOptions {
	return ActivityOptions{
		RetryPolicy: DefaultActivityRetryPolicy(),
		Timeout:     timeout,
	}
}

// WithHeartbeat creates ActivityOptions with heartbeat interval for long-running operations
// Heartbeat interval should be significantly shorter than the activity timeout
func WithHeartbeat(interval time.Duration) ActivityOptions {
	return ActivityOptions{
		RetryPolicy: DefaultActivityRetryPolicy(),
		Timeout:     AgentExecutionTimeoutPolicy(),
		Heartbeat:   interval,
	}
}

// ============================================================================
// ERROR CLASSIFICATION
// ============================================================================

// IsRetryableError determines if an error is transient and should trigger a retry
// Transient errors include:
// - Network timeouts
// - Temporary connection failures
// - Service unavailable (503)
// Non-retryable errors include:
// - Invalid input
// - File not found
// - Permission denied
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Timeout errors are retryable
	var timeoutErr interface{ Timeout() bool }
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return true
	}

	// Temporary errors are retryable
	var tempErr interface{ Temporary() bool }
	if errors.As(err, &tempErr) && tempErr.Temporary() {
		return true
	}

	// Service unavailable (503) is retryable
	if errMsg == "service unavailable" || errMsg == "temporarily unavailable" {
		return true
	}

	// Connection errors are retryable
	if errMsg == "connection refused" || errMsg == "connection reset" {
		return true
	}

	return false
}

// IsLockConflict checks if an error indicates a file lock conflict
// Lock conflict errors occur when:
// - File is already locked by another activity
// - Lock acquisition timeout
// - Concurrent modification detected
func IsLockConflict(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// File lock conflict patterns
	lockPatterns := []string{
		"file is locked",
		"lock conflict",
		"already locked",
		"lock acquisition failed",
		"locked by another process",
		"concurrent modification",
		"EACCES",
	}

	for _, pattern := range lockPatterns {
		if errMsg == pattern || len(errMsg) > len(pattern) && errMsg[:len(pattern)] == pattern {
			return true
		}
	}

	return false
}
