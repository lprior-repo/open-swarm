// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDefaultActivityRetryPolicy verifies standard retry configuration
func TestDefaultActivityRetryPolicy(t *testing.T) {
	policy := DefaultActivityRetryPolicy()

	assert.NotNil(t, policy)
	assert.Equal(t, int32(3), policy.MaximumAttempts)
	assert.Equal(t, 1*time.Second, policy.InitialInterval)
	assert.Equal(t, 2.0, policy.BackoffCoefficient)
	assert.Equal(t, 30*time.Second, policy.MaximumInterval)
}

// TestFileConflictRetryPolicy verifies lock-friendly retry configuration
func TestFileConflictRetryPolicy(t *testing.T) {
	policy := FileConflictRetryPolicy()

	assert.NotNil(t, policy)
	assert.Equal(t, int32(5), policy.MaximumAttempts)
	assert.Equal(t, 2*time.Second, policy.InitialInterval)
	assert.Equal(t, 2.0, policy.BackoffCoefficient)
	assert.Equal(t, 60*time.Second, policy.MaximumInterval)
}

// TestAgentExecutionTimeoutPolicy verifies 5-minute timeout
func TestAgentExecutionTimeoutPolicy(t *testing.T) {
	timeout := AgentExecutionTimeoutPolicy()
	assert.Equal(t, 5*time.Minute, timeout)
}

// TestWorkflowTimeoutPolicy verifies 30-minute timeout
func TestWorkflowTimeoutPolicy(t *testing.T) {
	timeout := WorkflowTimeoutPolicy()
	assert.Equal(t, 30*time.Minute, timeout)
}

// TestWithRetryPolicy creates options with custom retry policy
func TestWithRetryPolicy(t *testing.T) {
	customPolicy := DefaultActivityRetryPolicy()
	opts := WithRetryPolicy(customPolicy)

	assert.NotNil(t, opts.RetryPolicy)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
}

// TestWithTimeout creates options with custom timeout
func TestWithTimeout(t *testing.T) {
	customTimeout := 10 * time.Minute
	opts := WithTimeout(customTimeout)

	assert.NotNil(t, opts.RetryPolicy)
	assert.Equal(t, customTimeout, opts.Timeout)
}

// TestWithHeartbeat creates options with heartbeat interval
func TestWithHeartbeat(t *testing.T) {
	heartbeatInterval := 30 * time.Second
	opts := WithHeartbeat(heartbeatInterval)

	assert.NotNil(t, opts.RetryPolicy)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
	assert.Equal(t, heartbeatInterval, opts.Heartbeat)
}

// TestIsRetryableError_Timeout verifies timeout errors are retryable
func TestIsRetryableError_Timeout(t *testing.T) {
	// Create a mock timeout error
	timeoutErr := &net.DNSError{IsTimeout: true}
	assert.True(t, IsRetryableError(timeoutErr))
}

// TestIsRetryableError_ServiceUnavailable verifies 503-like errors are retryable
func TestIsRetryableError_ServiceUnavailable(t *testing.T) {
	err := errors.New("service unavailable")
	assert.True(t, IsRetryableError(err))

	err = errors.New("temporarily unavailable")
	assert.True(t, IsRetryableError(err))
}

// TestIsRetryableError_ConnectionErrors verifies connection errors are retryable
func TestIsRetryableError_ConnectionErrors(t *testing.T) {
	err := errors.New("connection refused")
	assert.True(t, IsRetryableError(err))

	err = errors.New("connection reset")
	assert.True(t, IsRetryableError(err))
}

// TestIsRetryableError_NonRetryable verifies non-retryable errors
func TestIsRetryableError_NonRetryable(t *testing.T) {
	// Invalid input - not retryable
	err := errors.New("invalid argument")
	assert.False(t, IsRetryableError(err))

	// File not found - not retryable
	err = errors.New("file not found")
	assert.False(t, IsRetryableError(err))

	// Permission denied - not retryable
	err = errors.New("permission denied")
	assert.False(t, IsRetryableError(err))
}

// TestIsRetryableError_NilError returns false for nil
func TestIsRetryableError_NilError(t *testing.T) {
	assert.False(t, IsRetryableError(nil))
}

// TestIsLockConflict_LockedByAnotherProcess detects lock conflicts
func TestIsLockConflict_LockedByAnotherProcess(t *testing.T) {
	err := errors.New("file is locked")
	assert.True(t, IsLockConflict(err))

	err = errors.New("locked by another process")
	assert.True(t, IsLockConflict(err))

	err = errors.New("already locked")
	assert.True(t, IsLockConflict(err))
}

// TestIsLockConflict_LockAcquisitionFailed detects acquisition failures
func TestIsLockConflict_LockAcquisitionFailed(t *testing.T) {
	err := errors.New("lock acquisition failed")
	assert.True(t, IsLockConflict(err))

	err = errors.New("lock conflict")
	assert.True(t, IsLockConflict(err))
}

// TestIsLockConflict_ConcurrentModification detects concurrent issues
func TestIsLockConflict_ConcurrentModification(t *testing.T) {
	err := errors.New("concurrent modification")
	assert.True(t, IsLockConflict(err))
}

// TestIsLockConflict_NonLockErrors returns false for non-lock errors
func TestIsLockConflict_NonLockErrors(t *testing.T) {
	err := errors.New("file not found")
	assert.False(t, IsLockConflict(err))

	err = errors.New("permission denied")
	assert.False(t, IsLockConflict(err))

	err = errors.New("connection failed")
	assert.False(t, IsLockConflict(err))
}

// TestIsLockConflict_NilError returns false for nil
func TestIsLockConflict_NilError(t *testing.T) {
	assert.False(t, IsLockConflict(nil))
}
