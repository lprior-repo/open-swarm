# Kill Switch Architecture

## Overview

The kill switch is a hierarchical failure propagation mechanism in the merge queue coordinator that immediately terminates all dependent speculative branches when a parent branch fails its tests. This prevents wasted computational resources on branches that are guaranteed to fail.

## Core Components

### 1. SpeculativeBranch Hierarchy

Speculative branches form a parent-child tree structure:

```
Branch A [C1]
  ├─ Branch B [C1, C2]
  │    └─ Branch D [C1, C2, C3]
  └─ Branch C [C1, C2, C3]
```

Each branch tracks:
- `ParentID`: ID of parent branch (empty for base branch)
- `ChildrenIDs`: IDs of child branches spawned from this one
- `Depth`: How many levels deep (1 = base, 2 = base+1, etc)

### 2. Kill Switch Functions

#### `killFailedBranch(ctx, branchID, reason) error`

Kills a single speculative branch and cleans up its resources.

**Usage Scenarios:**
1. **Direct failure**: When a branch's own tests fail
2. **Cascading kill**: When called by `killDependentBranches` due to parent failure

**Operations:**
- Sets branch Status to `BranchStatusKilled`
- Records `KilledAt` timestamp and `KillReason` for observability
- Increments `TotalKills` metric counter
- TODO: Cancels the Temporal workflow to stop test execution
- TODO: Stops and removes the Docker container
- TODO: Cleans up the git worktree

**Idempotency:**
- Killing an already-killed branch is safe and returns no error
- Original kill metadata (KilledAt, KillReason) is preserved on subsequent calls
- Prevents race conditions and duplicate cleanup attempts

**Thread Safety:**
- Acquires `c.mu` lock to safely modify activeBranches map
- Safe to call from multiple goroutines

**Error Handling:**
- Returns error only if branchID does not exist in activeBranches
- Resource cleanup failures are logged but not returned (TODO)

**Example:**
```go
// Direct kill due to test failure
err := c.killFailedBranch(ctx, "branch-42", "tests failed: timeout")

// Cascading kill due to parent failure
err := c.killFailedBranch(ctx, childID, fmt.Sprintf("parent branch %s failed", parentID))
```

#### `killDependentBranches(ctx, branchID) error`

Recursively kills all child branches when a parent fails.

**Algorithm:**
1. Locks to read the branch's `ChildrenIDs` list
2. Unlocks to avoid holding lock during recursion
3. For each child:
   a. Recursively kill the child's descendants (depth-first)
   b. Kill the child itself with reason "parent branch X failed"
4. Continues killing remaining branches even if errors occur

**Cascading Example:**

Given this hierarchy:
```
Branch A [C1]
  ├─ Branch B [C1, C2]
  │    └─ Branch D [C1, C2, C3]
  └─ Branch C [C1, C2, C3]
```

If Branch B fails:
- Branch D is killed (child of B)
- Branch A continues (parent of B)
- Branch C continues (sibling of B)

**Error Handling:**
- Errors killing individual branches are logged but don't stop the cascade
- This ensures maximum cleanup even when some operations fail
- Returns error only if the parent branchID is not found

**Thread Safety:**
- Temporarily acquires lock to read children list, then releases
- Recursively acquires lock for each child kill
- Designed to avoid deadlocks during deep recursion

**Example:**
```go
// Kill all descendants when a branch fails
if err := c.killDependentBranches(ctx, failedBranchID); err != nil {
    log.Warnf("Error killing dependent branches: %v", err)
}
```

### 3. BranchStatus States

```go
const (
    BranchStatusPending BranchStatus = "pending" // Waiting to start
    BranchStatusTesting BranchStatus = "testing" // Tests running
    BranchStatusPassed  BranchStatus = "passed"  // Tests passed
    BranchStatusFailed  BranchStatus = "failed"  // Tests failed
    BranchStatusKilled  BranchStatus = "killed"  // Killed by kill switch
)
```

**BranchStatusKilled** indicates the branch was terminated by the kill switch when a parent branch in the speculation hierarchy failed its tests.

## Integration with Test Results Processing

The kill switch is triggered in `processTestResult`:

```go
func (c *Coordinator) processTestResult(ctx context.Context, result *TestResult) {
    if result.Passed {
        // Merge successful changes
        c.mergeSuccessfulBranch(ctx, result)
    } else if failedBranchID != "" {
        // Kill switch activation
        // 1. Kill all dependent branches first
        c.killDependentBranches(ctx, failedBranchID)

        // 2. Kill the failed branch itself
        c.killFailedBranch(ctx, failedBranchID,
            fmt.Sprintf("tests failed: %s", result.ErrorMessage))
    }
}
```

## Performance Benefits

The kill switch provides significant performance improvements:

1. **Resource Savings**: Prevents wasted test execution on guaranteed failures
2. **Immediate Cleanup**: Frees Docker containers and CPU resources right away
3. **Reduced Latency**: Focuses queue on viable merge candidates
4. **Measurable Impact**: Tracked via `KilledPercent` metric in QueueStats

## Metrics

### Kill Switch Metrics in QueueStats

```go
type QueueStats struct {
    // KilledPercent is the percentage of branches terminated by the kill switch.
    // High values indicate effective resource savings from hierarchical failure propagation.
    KilledPercent float64

    // TotalKills counts how many branches were terminated via killFailedBranch.
    // This includes both direct failures and cascading kills from parent failures.
    TotalKills    int64

    TotalFailures int64 // Total number of test failures
    TotalTimeouts int64 // Total number of test timeouts
    // ... other metrics
}
```

**Interpreting Metrics:**

- **High KilledPercent (>30%)**: Kill switch is actively saving resources by terminating failing speculation chains
- **Low KilledPercent (<10%)**: Either high test pass rates or shallow speculation depth
- **TotalKills >> TotalFailures**: Effective cascading kills preventing wasted work
- **TotalKills ≈ TotalFailures**: Mostly shallow hierarchies or isolated failures

## Configuration

### KillSwitchTimeout

```go
type CoordinatorConfig struct {
    // Kill switch timeout (default 500ms)
    // Maximum time to wait for kill operations to complete
    KillSwitchTimeout time.Duration
    // ...
}
```

The `KillSwitchTimeout` controls how long the coordinator waits for resource cleanup operations (Temporal workflow cancellation, Docker container stops, etc.) to complete before proceeding.

**Default**: 500ms

**Tuning Guidelines:**
- Increase if you have slow Docker operations or complex workflow cleanup
- Decrease for faster failure recovery in high-throughput scenarios
- Monitor kill operation duration to optimize this value

## Implementation Status

### Completed ✓
- [x] Hierarchical branch tracking (ParentID, ChildrenIDs)
- [x] Idempotent killFailedBranch function
- [x] Recursive killDependentBranches function
- [x] BranchStatusKilled state
- [x] Kill metadata (KilledAt, KillReason)
- [x] TotalKills metric tracking
- [x] Integration with test result processing

### TODO
- [ ] Temporal workflow cancellation on kill
- [ ] Docker container cleanup on kill
- [ ] Git worktree cleanup on kill
- [ ] Kill operation timeout enforcement
- [ ] Kill event logging and observability
- [ ] KilledPercent metric calculation

## Testing

Kill switch behavior is tested in `coordinator_test.go`:

### Test Coverage

1. **TestKillFailedBranch_Success**: Verifies basic kill operation
2. **TestKillFailedBranch_Idempotent**: Verifies idempotency guarantees
3. **TestKillFailedBranch_NonExistentBranch**: Error handling for invalid IDs
4. **TestKillFailedBranch_DifferentStatuses**: Kill from any status works
5. **TestKillFailedBranch_PreservesOtherBranches**: No accidental side effects

### Future Test Needs

- [ ] Multi-level hierarchy kill cascades (3+ levels deep)
- [ ] Concurrent kill operations (race condition testing)
- [ ] Kill during active test execution
- [ ] Resource cleanup verification
- [ ] Metric accuracy under various failure patterns

## Example Usage

### Creating Speculative Branches with Hierarchy

```go
func (c *Coordinator) createSpeculativeBranches(ctx context.Context, batch []*ChangeRequest) {
    for depth := 1; depth <= len(batch); depth++ {
        changes := batch[:depth]
        branchID := c.generateBranchID(changes)

        // Determine parent (previous depth level)
        var parentID string
        if depth > 1 {
            parentChanges := batch[:depth-1]
            parentID = c.generateBranchID(parentChanges)
        }

        branch := &SpeculativeBranch{
            ID:          branchID,
            Changes:     changes,
            Depth:       depth,
            ParentID:    parentID,
            ChildrenIDs: []string{},
        }

        // Update parent's children list
        if parentID != "" {
            parent := c.activeBranches[parentID]
            parent.ChildrenIDs = append(parent.ChildrenIDs, branchID)
        }

        c.activeBranches[branchID] = branch
        go c.executeSpeculativeBranch(ctx, branch)
    }
}
```

### Triggering Kill Switch on Failure

```go
func (c *Coordinator) processTestResult(ctx context.Context, result *TestResult) {
    if !result.Passed {
        // Find the failed branch
        failedBranchID := c.findBranchByResult(result)

        // Cascade kill through hierarchy
        if err := c.killDependentBranches(ctx, failedBranchID); err != nil {
            log.Warnf("Error killing dependents: %v", err)
        }

        // Kill the failed branch
        if err := c.killFailedBranch(ctx, failedBranchID,
            fmt.Sprintf("tests failed: %s", result.ErrorMessage)); err != nil {
            log.Errorf("Error killing failed branch: %v", err)
        }
    }
}
```

## Related Documentation

- See `SpeculativeBranch` type documentation in `types.go`
- See Uber-style merge queue design in main README
- See Temporal workflow integration docs (TODO)
- See Docker container management docs (TODO)
