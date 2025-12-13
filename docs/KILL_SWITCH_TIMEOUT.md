# Kill Switch Timeout Implementation

## Overview

This document describes the timeout handling implementation for kill switch operations in the merge queue coordinator.

## Problem

Previously, the `killFailedBranch` and `killDependentBranches` functions did not enforce the configured `KillSwitchTimeout`, which could lead to:
- Indefinite hangs if cleanup operations (Temporal workflow cancellation, Docker container stopping, worktree cleanup) get stuck
- Resource leaks when kill operations don't complete
- Cascading failures in deep branch hierarchies

## Solution

### New Implementation

We've added timeout-aware versions of the kill functions in `/internal/mergequeue/kill_switch.go`:

1. **`killFailedBranchWithTimeout`** - Timeout-aware single branch kill operation
2. **`killDependentBranchesWithTimeout`** - Timeout-aware recursive cascade kill operation

### Key Features

#### 1. Timeout Enforcement

Both functions create timeout contexts using the configured `KillSwitchTimeout`:

```go
killCtx, cancel := context.WithTimeout(ctx, c.config.KillSwitchTimeout)
defer cancel()
```

#### 2. Graceful Degradation

If cleanup operations time out, the branch is still marked as killed to ensure consistency:

```go
case <-killCtx.Done():
    // Graceful degradation: mark as killed anyway even if cleanup timed out
    c.mu.Lock()
    if branch, exists := c.activeBranches[branchID]; exists && branch.Status != BranchStatusKilled {
        now := time.Now()
        branch.Status = BranchStatusKilled
        branch.KilledAt = &now
        branch.KillReason = fmt.Sprintf("%s (timeout during cleanup)", reason)
        c.stats.TotalKills++
    }
    c.mu.Unlock()
    return fmt.Errorf("kill operation timed out after %v (branch marked as killed)", c.config.KillSwitchTimeout)
```

#### 3. Idempotency Preserved

The timeout-aware functions maintain the same idempotency guarantees:
- Killing an already-killed branch returns success without modifying state
- Original kill metadata (KilledAt timestamp, KillReason) is preserved

#### 4. Cascade Timeout Handling

For deep branch hierarchies, `killDependentBranchesWithTimeout` uses a multiplied timeout:

```go
// Use a multiple of KillSwitchTimeout to allow for deep hierarchies
cascadeTimeout := c.config.KillSwitchTimeout * 10
```

This ensures that even deep hierarchies have sufficient time to complete the cascade operation.

#### 5. Partial Progress Tracking

If a cascade operation times out:
- Already-killed branches remain killed
- Un-killed branches remain in their current state
- First error encountered is returned
- System remains in a consistent state

## Configuration

The timeout is controlled by the `KillSwitchTimeout` field in `CoordinatorConfig`:

```go
config := CoordinatorConfig{
    KillSwitchTimeout: 500 * time.Millisecond, // Default: 500ms
    // ... other config
}
```

## Usage

### Killing a Single Branch with Timeout

```go
err := coordinator.killFailedBranchWithTimeout(ctx, branchID, "test failure")
if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        // Branch is marked as killed, but cleanup may not be complete
        log.Warn("Kill operation timed out", "branchID", branchID, "error", err)
    } else {
        // Other error (e.g., branch not found)
        log.Error("Failed to kill branch", "branchID", branchID, "error", err)
    }
}
```

### Killing Branch Hierarchy with Timeout

```go
err := coordinator.killDependentBranchesWithTimeout(ctx, parentBranchID)
if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        // Partial progress - some branches may be killed
        log.Warn("Kill cascade timed out", "parentID", parentBranchID, "error", err)
    } else {
        // Other error
        log.Error("Failed to kill dependent branches", "parentID", parentBranchID, "error", err)
    }
}
```

## Testing

Comprehensive test coverage is provided in `/internal/mergequeue/kill_switch_test.go`:

### Test Cases

1. **Success Cases**
   - `TestKillFailedBranchWithTimeout_Success` - Normal kill operation completes quickly
   - `TestKillDependentBranchesWithTimeout_SimpleHierarchy` - Parent with 2 children
   - `TestKillDependentBranchesWithTimeout_DeepHierarchy` - 3-level deep hierarchy
   - `TestKillDependentBranchesWithTimeout_MultipleChildren` - Complex tree structure

2. **Timeout Cases**
   - `TestKillFailedBranchWithTimeout_GracefulDegradation` - Branch marked as killed despite timeout
   - `TestKillDependentBranchesWithTimeout_Timeout` - Cascade times out with partial progress

3. **Edge Cases**
   - `TestKillFailedBranchWithTimeout_Idempotent` - Killing already-killed branch
   - `TestKillFailedBranchWithTimeout_NonExistentBranch` - Error handling
   - `TestKillDependentBranchesWithTimeout_AlreadyKilledChildren` - Mixed state handling

4. **Consistency Cases**
   - `TestKillFailedBranchWithTimeout_PreservesOtherBranches` - Isolation verification
   - `TestKillFailedBranchWithTimeout_MetricsTracking` - Counter accuracy

## Metrics

The timeout implementation tracks:
- `TotalKills` - Incremented for each successful kill (including timeout with graceful degradation)
- Kill operations return error on timeout, allowing caller to track timeout frequency

## Future Enhancements

1. **Add timeout metrics** - Track how often kill operations timeout
2. **Adaptive timeout** - Adjust timeout based on historical cleanup duration
3. **Background cleanup** - For timed-out operations, continue cleanup in background
4. **Cleanup validation** - Verify resources are actually freed after timeout

## Migration

The original `killFailedBranch` and `killDependentBranches` functions remain in `coordinator.go` for backward compatibility. New code should use the timeout-aware versions:

```go
// Old (no timeout enforcement)
err := c.killFailedBranch(ctx, branchID, reason)

// New (with timeout enforcement)
err := c.killFailedBranchWithTimeout(ctx, branchID, reason)
```

To fully migrate, update all call sites to use the timeout-aware functions, then consider deprecating or removing the old functions.

## Related Issues

- `open-swarm-ssht` - Timeout handling for kill switch operations (this implementation)
- `open-swarm-ltf6` - Unit tests for killDependentBranches with hierarchy (covered by tests)
- `open-swarm-ho1g` - killDependentBranches recursive killer (completed)

## Implementation Files

- `/internal/mergequeue/kill_switch.go` - Timeout-aware kill functions
- `/internal/mergequeue/kill_switch_test.go` - Comprehensive test suite
- `/internal/mergequeue/coordinator.go` - Original functions (backward compatible)
- `/internal/mergequeue/coordinator_test.go` - Original tests
