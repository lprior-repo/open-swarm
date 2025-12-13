# Kill Switch Error Handling & Resilience Improvements

## Summary

This document describes comprehensive improvements made to the branch kill functionality in the merge queue coordinator to add robust error handling, retry logic, rollback mechanisms, and clear error messages.

## Files Added

### 1. `/internal/mergequeue/kill_switch_errors.go`
Defines custom error types for kill switch operations:

**KillSwitchError**: Wraps errors with operation context
- `Operation`: Name of the kill operation (kill_branch, cleanup_workflow, notify_agents)
- `BranchID`: ID of branch being killed
- `Err`: Underlying error
- `RetryCount`: Number of retry attempts
- `Recoverable`: Whether the operation can be safely retried
- `BranchMarkedKilled`: Whether branch was marked killed despite the error (graceful degradation)
- `Context`: Additional context about what failed

**CleanupError**: Represents resource cleanup failures
- `ResourceType`: Type of resource (workflow, container, worktree, notification)
- `ResourceID`: Identifier of resource
- `Operation`: Operation that failed (cancel, stop, remove, send)
- `Err`: Underlying error
- `Retryable`: Whether cleanup can be retried
- `CanDegrade`: Whether operation can gracefully degrade

**TimeoutError**: Represents kill operations that timed out
- `Step`: Which step timed out
- `BranchID`: Branch being killed
- `ConfiguredTimeout`: Timeout in milliseconds
- `PartialProgress`: Whether partial progress was made before timeout
- `CompletedSteps`: Steps that completed
- `PendingSteps`: Steps that didn't complete

**UserFacingError**: Formatted errors for user display
- `Title`: Brief issue title
- `Message`: User-friendly explanation
- `SuggestedActions`: Actions to resolve the issue
- `TechnicalDetails`: Technical details for debugging
- `BranchID`: Branch reference

**RetryConfig**: Configuration for retry logic
- `MaxRetries`: Maximum retry attempts (default 3)
- `InitialDelayMs`: Initial delay between retries (default 100ms)
- `MaxDelayMs`: Maximum delay between retries (default 5000ms)
- `BackoffMultiplier`: Backoff multiplier for exponential backoff (default 2.0)
- `JitterPercent`: Jitter percentage (default 10%)

### 2. `/internal/mergequeue/kill_switch_recovery.go`
Implements retry logic and recovery mechanisms:

**RetryStrategy**: Manages retry logic with exponential backoff
- `ShouldRetry()`: Determines if operation should be retried
- `NextRetryDelay()`: Calculates delay before next retry with jitter
- `RecordStep()`: Tracks completed steps for recovery
- `RecordError()`: Records errors and increments retry counter
- `GetState()`: Returns current operation state

**ExecuteWithRetry()**: Helper function for retrying operations
- Automatic retry with exponential backoff
- Context-aware timeout handling
- Structured error wrapping

**KillSwitchRecoveryManager**: Manages rollback and recovery
- `RegisterRollback()`: Register rollback actions in LIFO order
- `LogOperation()`: Log operations for debugging
- `Rollback()`: Execute all rollback actions in reverse order
- `SaveBranchState()`: Create state snapshot before modifications
- `GetOperationLog()`: Get operation log for debugging

**CascadeKillValidator**: Prevents resource exhaustion
- `CanCascade()`: Check cascade safety before proceeding
- `EnterBranch()`: Mark branch as being processed
- `ExitBranch()`: Mark branch as no longer being processed
- Prevents circular references and excessive depth
- Limits total branches killed per cascade (default 1000)
- Limits recursion depth (default 20)

### 3. `/internal/mergequeue/KILL_SWITCH_ERROR_HANDLING.md`
Comprehensive documentation covering:
- Error handling strategies for single branch kills and cascade kills
- Retry logic implementation
- Rollback and recovery mechanisms
- Graceful degradation principles
- Notification error handling
- Clear error message formatting
- Best practices for using the error types
- Examples of error handling patterns

## Files Modified

### `/internal/mergequeue/kill_switch.go`

**Enhanced killFailedBranchWithTimeout()**:
1. Improved error wrapping using KillSwitchError
   - Provides operation context
   - Indicates recoverability
   - Includes detailed context information

2. Better timeout handling
   - Returns TimeoutError instead of generic error
   - Includes completed/pending step information
   - Indicates partial progress (branch marked as killed)

3. Race condition protection documentation
   - Explains idempotency checks
   - Ensures atomic state transitions
   - Prevents duplicate metric increments

**Enhanced killDependentBranchesRecursive()**:
1. Improved error wrapping for not-found errors
   - Structured KillSwitchError with context

2. Better cascade error handling
   - UserFacingError wrapping for child failures
   - Suggested actions for remediation
   - Partial success tracking (killed vs failed counts)

3. Comprehensive timeout reporting
   - TimeoutError with progress metrics
   - Completed vs pending step information
   - Partial progress indication

4. Error aggregation
   - Captures first error
   - Wraps with KillSwitchError if partial success
   - Provides context about success/failure ratio

## Error Handling Improvements

### Before
```go
// Generic error messages with no context
if !exists {
    return fmt.Errorf("branch %s not found", branchID)
}

// Timeout errors are indistinguishable from other errors
return fmt.Errorf("kill operation timed out after %v (branch marked as killed)", timeout)

// No indication of partial success or progress
return fmt.Errorf("kill cascade timed out while processing children")
```

### After
```go
// Structured errors with full context
if !exists {
    return &KillSwitchError{
        Operation:   "kill_single_branch",
        BranchID:    branchID,
        Err:         fmt.Errorf("branch not found"),
        Recoverable: false,
        Context:     "branch does not exist in active branches",
    }
}

// Clear timeout errors with progress information
return &TimeoutError{
    Step:              "kill_single_branch",
    BranchID:          branchID,
    ConfiguredTimeout: c.config.KillSwitchTimeout.Milliseconds(),
    PartialProgress:   true,
    CompletedSteps:    []string{"marked_as_killed"},
    PendingSteps:      []string{"cleanup", "notification"},
}

// Detailed cascade error reporting
return &TimeoutError{
    Step:              "cascade_kill",
    BranchID:          branchID,
    PartialProgress:   successCount > 0,
    CompletedSteps:    []string{fmt.Sprintf("killed %d children", successCount)},
    PendingSteps:      []string{fmt.Sprintf("%d remaining", len(childrenIDs)-successCount-failureCount)},
}
```

## Key Features Implemented

### 1. Proper Error Wrapping and Context
- Custom error types provide operation context
- Each error includes the branch ID being killed
- Errors indicate whether they're recoverable
- Additional context explains what went wrong

### 2. Retry Logic for Transient Failures
- RetryStrategy with exponential backoff
- Configurable max retries, delays, and jitter
- Distinguishes between transient and permanent failures
- `ExecuteWithRetry()` helper for common retry patterns

### 3. Rollback/Cleanup on Failure
- KillSwitchRecoveryManager tracks rollback actions
- LIFO stack for proper rollback order
- State snapshots before modifications
- Operation logging for recovery and debugging

### 4. Clear Error Messages for Users
- UserFacingError with actionable guidance
- Suggested actions for remediation
- Technical details for debugging
- Clear titles and descriptions

### 5. Graceful Degradation
- Branch always marked as killed, even if cleanup times out
- Metrics always updated for completed kills
- Timeout errors still indicate success with warnings
- Resources can be garbage collected later

### 6. Cascade Safety
- CascadeKillValidator prevents infinite recursion
- Maximum recursion depth limit (default 20)
- Maximum branches per cascade limit (default 1000)
- Circular reference detection

### 7. Comprehensive Error Tracking
- Success/failure counts for cascades
- Completed vs pending step tracking
- Partial progress indication
- Wrapped errors with added context

## Usage Examples

### Handling Kill Errors
```go
err := c.killFailedBranch(ctx, branchID, "tests failed")
switch err.(type) {
case *TimeoutError:
    // Branch marked as killed, cleanup may be pending
    logger.Warn("Kill timed out", "error", err)
case *KillSwitchError:
    killErr := err.(*KillSwitchError)
    if killErr.Recoverable {
        logger.Info("Safe to retry", "error", err)
    } else {
        logger.Error("Unrecoverable error", "error", err)
    }
case *UserFacingError:
    // Return to user with suggested actions
    return err
default:
    logger.Error("Unknown error", "error", err)
}
```

### Cascade Kill Error Handling
```go
err := c.killDependentBranches(ctx, failedBranch)
if err != nil {
    if userErr, ok := err.(*UserFacingError); ok {
        // Already formatted for user display
        logger.Error("Cascade failed",
            "error", userErr.Message,
            "suggestions", userErr.SuggestedActions)
    } else if killErr, ok := err.(*KillSwitchError); ok {
        // Log with context about partial success
        logger.Error("Partial cascade failure",
            "context", killErr.Context,
            "error", killErr.Err)
    }
}
```

## Benefits

1. **Better Observability**
   - Structured errors enable better logging and monitoring
   - Operation context helps with debugging
   - Progress tracking shows what was completed

2. **Improved Resilience**
   - Graceful degradation ensures consistency
   - Retry logic handles transient failures
   - Cascade safety prevents resource exhaustion

3. **Better User Experience**
   - Clear error messages with actionable guidance
   - Suggested fixes for common problems
   - Technical details for advanced users

4. **Easier Maintenance**
   - Comprehensive documentation
   - Well-defined error types
   - Recovery mechanisms for failures

## Testing Recommendations

### Unit Tests
- Test each error type construction
- Test RetryStrategy backoff calculations
- Test RecoveryManager rollback sequences
- Test CascadeKillValidator limits

### Integration Tests
- Test timeout scenarios with partial progress
- Test cascade kills with mixed failures
- Test concurrent kill attempts (race conditions)
- Test graceful degradation in resource cleanup

### Chaos Tests
- Kill operations with flaky networks
- Docker container failures
- Temporal workflow cancellation failures
- Notification service outages

## Future Improvements

1. **Metrics Integration**
   - Track retry attempts per branch
   - Monitor timeout frequency
   - Measure rollback frequency

2. **Adaptive Retry Strategies**
   - Adjust timeout values based on historical success rates
   - Dynamic backoff based on system load

3. **Resource Cleanup Tracking**
   - Monitor Temporal workflow cleanup status
   - Track Docker container cleanup
   - Audit notification delivery

4. **Agent Notifications**
   - Include error details in kill notifications
   - Suggested actions for users
   - Recovery status updates

## Conclusion

The kill switch improvements provide a solid foundation for reliable branch failure handling in the merge queue. The structured error types, retry logic, and graceful degradation mechanisms ensure the system remains resilient under failure conditions while providing clear feedback to users and operators.
