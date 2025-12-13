# Kill Switch Error Handling & Recovery

## Overview

This document describes the comprehensive error handling, retry logic, rollback mechanisms, and recovery strategies implemented for the kill switch functionality in the merge queue coordinator.

## Error Types

### 1. KillSwitchError
Represents errors that occur during kill switch operations with context about whether the operation was recoverable.

```go
type KillSwitchError struct {
    Operation string          // "kill_branch", "cleanup_workflow", "notify_agents"
    BranchID string           // Branch being killed
    Err error                 // Underlying error
    RetryCount int           // Number of retry attempts
    Recoverable bool         // Whether the operation can be retried
    BranchMarkedKilling bool // Whether graceful degradation occurred
    Context string           // Additional context
}
```

**Usage**: When a kill switch operation fails, wrap the error as KillSwitchError to provide context about the operation, branch, and recoverability.

### 2. CleanupError
Represents failures during resource cleanup (Temporal workflows, Docker containers, worktrees, notifications).

```go
type CleanupError struct {
    ResourceType string  // "workflow", "container", "worktree", "notification"
    ResourceID string    // Identifier of the resource
    Operation string     // "cancel", "stop", "remove", "send"
    Err error           // Underlying error
    Retryable bool      // Can the cleanup be retried?
    CanDegrade bool     // Can the operation gracefully degrade?
}
```

**Usage**: When any resource cleanup fails, wrap as CleanupError to classify the failure and indicate retry/degrade options.

### 3. TimeoutError
Represents kill operations that exceeded their timeout.

```go
type TimeoutError struct {
    Step string                   // Which step timed out
    BranchID string              // Branch being killed
    ConfiguredTimeout int64      // Timeout in milliseconds
    PartialProgress bool         // Was partial progress made?
    CompletedSteps []string      // Steps that completed
    PendingSteps []string        // Steps that didn't complete
}
```

**Usage**: When a kill operation times out, return TimeoutError to indicate that the branch was marked as killed despite the timeout (graceful degradation).

### 4. UserFacingError
Formatted error for user display with suggested remediation actions.

```go
type UserFacingError struct {
    Title string              // Brief issue title
    Message string            // User-friendly explanation
    SuggestedActions []string // Actions to resolve
    TechnicalDetails string   // Details for debugging
    BranchID string          // For reference
}
```

**Usage**: When cascading errors or communicating with users, wrap as UserFacingError to provide actionable guidance.

## Error Handling Strategies

### 1. Single Branch Kill (killFailedBranchWithTimeout)

**Error Flow**:
1. Acquire lock and read branch state
2. Save branch state snapshot for potential rollback
3. Register rollback actions in LIFO order
4. Execute kill operation with timeout
5. On timeout: Mark branch as killed anyway (graceful degradation)
6. Return appropriate error type

**Key Features**:
- Timeout enforcement via context
- Graceful degradation: branch is marked killed even if cleanup times out
- Operation logging for recovery and debugging
- Rollback capability for critical errors

**Error Handling Examples**:
```go
// Branch not found - unrecoverable
return &KillSwitchError{
    Operation: "kill_single_branch",
    BranchID: branchID,
    Err: fmt.Errorf("branch not found"),
    Recoverable: false,
}

// Timeout with graceful degradation
return &TimeoutError{
    Step: "kill_single_branch",
    BranchID: branchID,
    ConfiguredTimeout: timeout.Milliseconds(),
    PartialProgress: true,
    CompletedSteps: []string{"marked_as_killed"},
    PendingSteps: []string{"cleanup", "notification"},
}
```

### 2. Cascade Kill (killDependentBranchesWithTimeout)

**Error Flow**:
1. Create cascade validator for safety
2. Validate cascade depth and branch count limits
3. Recursively kill children with timeout
4. Track partial success/failure
5. Return comprehensive error if any child kill failed

**Key Features**:
- Cascade safety validation (max depth, max branches)
- Concurrent processing capability
- Partial failure handling: continue killing other branches
- Comprehensive error reporting with counts

**Error Handling Strategy**:
```go
// Capture first error but continue processing
var firstErr error

for _, childID := range childrenIDs {
    // Check timeout
    select {
    case <-ctx.Done():
        return &TimeoutError{
            Step: "cascade_kill",
            PartialProgress: successCount > 0,
            CompletedSteps: [...completed...],
            PendingSteps: [...pending...],
        }
    default:
    }
    
    // Kill child
    if err := c.killFailedBranchWithTimeout(ctx, childID, reason); err != nil {
        if firstErr == nil {
            firstErr = &UserFacingError{
                Title: "Cascade Kill Failed",
                Message: fmt.Sprintf("Failed to kill %s", childID),
                SuggestedActions: [...],
                TechnicalDetails: err.Error(),
            }
        }
        continue  // Process other children
    }
    
    successCount++
}

// Return error with context about partial success
if firstErr != nil && successCount > 0 {
    firstErr = &KillSwitchError{
        Operation: "cascade_kill",
        Err: firstErr,
        Context: fmt.Sprintf("partial: %d killed, %d failed", successCount, failureCount),
    }
}
return firstErr
```

## Retry Logic

### RetryStrategy

The RetryStrategy class manages retry logic with exponential backoff and jitter.

```go
type RetryConfig struct {
    MaxRetries int            // 0 = no retries (default 3)
    InitialDelayMs int        // 100ms (default)
    MaxDelayMs int           // 5000ms (default)
    BackoffMultiplier float64 // 2.0 (exponential backoff)
    JitterPercent int         // 10% (add randomness)
}

// Usage
config := DefaultRetryConfig()
strategy := NewRetryStrategy(branchID, reason, config)

for {
    err := killOperation()
    if err == nil {
        return nil
    }
    
    strategy.RecordError(err)
    if !strategy.ShouldRetry() {
        return &KillSwitchError{
            Operation: "...",
            RetryCount: strategy.state.RetryAttempt,
            Recoverable: false,
        }
    }
    
    delay := strategy.NextRetryDelay()
    select {
    case <-time.After(delay):
        // Retry
    case <-ctx.Done():
        return fmt.Errorf("retry cancelled: %w", ctx.Err())
    }
}
```

### Transient vs Permanent Failures

**Transient Failures** (retryable):
- Network timeouts
- Resource unavailable temporarily
- Workflow/container not responding
- Notification service temporarily down

**Permanent Failures** (not retryable):
- Branch not found
- Invalid branch state
- Authorization/permission errors
- Circular cascade detected

## Rollback & Recovery

### KillSwitchRecoveryManager

Manages rollback operations for kill failures.

```go
type KillSwitchRecoveryManager struct {
    branchID string
    rollbackStack []*RollbackAction  // LIFO stack
    operationLog []string             // For debugging
}

// Usage
recovery := NewKillSwitchRecoveryManager(branchID)

// Record state before modifications
recovery.SaveBranchState(branch)

// Register rollback actions (executed in reverse order)
recovery.RegisterRollback("revert branch status", func(ctx context.Context) error {
    // Undo the status change
    branch.Status = BranchStatusFailed
    branch.KilledAt = nil
    branch.KillReason = ""
    return nil
})

// On failure, roll back
if err != nil {
    rollbackResult := recovery.Rollback(ctx)
    if !rollbackResult.Success {
        // Rollback had errors
        for _, err := range rollbackResult.Errors {
            logger.Error("Rollback failed", "error", err)
        }
    }
}
```

**Note**: Most kill operations cannot be rolled back due to graceful degradation. Rollback is reserved for critical pre-kill validations.

## Cascade Kill Validation

### CascadeKillValidator

Prevents infinite recursion and resource exhaustion during cascade kills.

```go
type CascadeKillValidator struct {
    MaxDepth int                    // 20 (max recursion depth)
    MaxBranchesPerCascade int       // 1000 (max branches to kill)
    ProcessingBranches map[string]bool  // Circular ref detection
    CurrentDepth int                // Current recursion level
}

// Usage
validator := NewCascadeKillValidator(20, 1000)

if err := validator.CanCascade(branchID); err != nil {
    return &KillSwitchError{
        Operation: "cascade_kill",
        BranchID: branchID,
        Err: fmt.Errorf("cascade limit exceeded"),
        Recoverable: false,
    }
}

validator.EnterBranch(branchID)
defer validator.ExitBranch(branchID)

// ... perform recursive kill ...
```

**Limits**:
- **MaxDepth**: Prevents stack overflow from circular references or very deep hierarchies
- **MaxBranchesPerCascade**: Prevents resource exhaustion from killing too many branches at once

## Graceful Degradation

### Principle

The kill switch prioritizes **state consistency** over **complete resource cleanup**:

1. **Always** mark the branch as killed (status update)
2. **Always** record kill timestamp and reason
3. **Always** update metrics (TotalKills)
4. **If possible** clean up resources (Temporal, Docker, notifications)
5. **If timeout/error**: Return error but maintain consistency

### Implementation

```go
// Attempt kill with timeout
select {
case result := <-resultChan:
    return result.err  // Success

case <-killCtx.Done():
    // Timeout occurred - graceful degradation
    
    c.mu.Lock()
    if branch, exists := c.activeBranches[branchID]; exists {
        now := time.Now()
        branch.Status = BranchStatusKilled     // ALWAYS update
        branch.KilledAt = &now
        branch.KillReason = fmt.Sprintf("%s (timeout during cleanup)", reason)
        c.stats.TotalKills++
    }
    c.mu.Unlock()
    
    // Return error but branch is marked as killed
    return &TimeoutError{
        Step: "kill_single_branch",
        BranchID: branchID,
        PartialProgress: true,  // State updated, cleanup may be pending
    }
}
```

## Notification Error Handling

### Non-blocking Notifications

Branch kill notifications are non-blocking: notification failures do not fail the kill operation.

```go
// Kill is successful even if notification fails
c.mu.Unlock()
notifyErr := c.notifyBranchKilled(ctx, branch, reason)
if notifyErr != nil {
    logger.Warn("Notification failed but kill succeeded",
        "branch_id", branchID,
        "notification_error", notifyErr.Error(),
    )
}
c.mu.Lock()
```

This ensures that:
- Agents may not receive notifications if the notification service is down
- The merge queue continues to function
- Notifications can be retried or reconciled later

## Clear Error Messages

### User-Facing Errors

All errors returned to users include:

1. **Clear Title**: What went wrong
2. **Detailed Message**: Why it happened
3. **Suggested Actions**: How to fix it
4. **Technical Details**: For debugging

```go
&UserFacingError{
    Title: "Cascade Kill Failed",
    Message: "Failed to kill child branch branch-123 and its descendants",
    SuggestedActions: []string{
        "Check system logs for resource cleanup issues",
        "Verify Temporal workflow and Docker container status",
        "Retry the kill operation manually if needed",
    },
    TechnicalDetails: "Original error: connection refused to Temporal server",
    BranchID: branchID,
}
```

### Error Logging

All kill operations are logged with:
- Timestamp
- Branch ID
- Operation type
- Reason
- Duration
- Success/failure status
- Error details if applicable

```go
logger.Info("Kill switch completed",
    "branch_id", branchID,
    "reason", reason,
    "killed_at", now,
    "total_kills", stats.TotalKills,
    "duration_ms", duration.Milliseconds(),
)

logger.Error("Cascade kill failed",
    "parent_branch_id", branchID,
    "total_children", len(childrenIDs),
    "failed_kills", failureCount,
    "first_error", firstErr.Error(),
)
```

## Best Practices

### When Implementing Kill Operations

1. **Always save state before modification**: Use RecoveryManager.SaveBranchState()
2. **Register rollback actions**: In LIFO order (latest first)
3. **Check context before each iteration**: Handle timeouts gracefully
4. **Continue on child errors**: Don't stop cascade due to one child failure
5. **Wrap errors appropriately**: Use KillSwitchError, CleanupError, TimeoutError, UserFacingError
6. **Log comprehensively**: Use slog for structured logging
7. **Update metrics**: Always increment counters for tracking

### When Catching Kill Errors

```go
if err := c.killFailedBranch(ctx, branchID, reason); err != nil {
    // Handle different error types
    switch err.(type) {
    case *TimeoutError:
        // Branch still marked as killed, cleanup may be pending
        logger.Warn("Kill timed out but branch was marked as killed", "error", err)
        
    case *KillSwitchError:
        if err.(*KillSwitchError).Recoverable {
            // Safe to retry
            logger.Info("Recoverable error, safe to retry", "error", err)
        } else {
            // Permanent failure
            logger.Error("Unrecoverable error", "error", err)
        }
        
    case *UserFacingError:
        // Return to user with suggestions
        return err
        
    default:
        logger.Error("Unknown error type", "error", err)
    }
}
```

## Summary

The kill switch error handling provides:

1. **Structured Error Types**: KillSwitchError, CleanupError, TimeoutError, UserFacingError
2. **Retry Logic**: Exponential backoff with jitter for transient failures
3. **Rollback Capability**: LIFO stack for rolling back changes on failure
4. **Graceful Degradation**: Branch always marked as killed, even if cleanup times out
5. **Cascade Safety**: Validator prevents infinite recursion and resource exhaustion
6. **Clear Error Messages**: Actionable guidance for users and operators
7. **Comprehensive Logging**: Structured logs for monitoring and debugging
8. **Non-blocking Notifications**: Kill succeeds even if agent notification fails

These mechanisms ensure the merge queue remains resilient and consistent under failure conditions.
