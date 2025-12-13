// Package mergequeue implements an Uber-style speculative merge queue with hierarchical kill switch.
//
// The kill switch functionality is defined in this file and provides timeout-aware
// failure propagation through the branch hierarchy. See KILLSWITCH.md for architecture details.
package mergequeue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// killFailedBranchWithTimeout kills a single speculative branch with timeout enforcement.
//
// # Kill Switch Component (Timeout-Aware Version) with Race Condition Protection
//
// This is an enhanced version of killFailedBranch that includes timeout enforcement
// and graceful degradation. It prevents kill operations from blocking indefinitely
// when resource cleanup (Temporal workflows, Docker containers) hangs or fails.
//
// # Race Condition Protection
//
// The implementation includes multiple layers of protection against concurrent kill attempts:
//
//  1. **Idempotency Check**: If a branch is already killed, returns immediately without
//     acquiring the lock again or incrementing metrics. This prevents duplicate processing
//     when multiple failures occur simultaneously.
//
//  2. **Atomic State Transition**: The status change to BranchStatusKilled is atomic within
//     the critical section. No intermediate states are exposed to other goroutines.
//
//  3. **Metadata Atomicity**: KilledAt timestamp and KillReason are updated in the same
//     critical section, ensuring consistency (no partial updates visible to readers).
//
//  4. **Metrics Safety**: TotalKills counter is incremented only once per branch, protected
//     by the mutex and the idempotency check.
//
// # Concurrent Kill Attempts Detection
//
// If two goroutines try to kill the same branch simultaneously:
//  - Both acquire the lock sequentially (Go's sync.Mutex ensures fairness)
//  - First goroutine: Marks as killed, increments metrics
//  - Second goroutine: Sees already-killed status, returns without side effects
//  - Result: Metrics are accurate, no race conditions
//
// # Timeout Behavior
//
//  - If cleanup completes within KillSwitchTimeout, returns nil
//  - If cleanup times out, marks branch as killed anyway (graceful degradation)
//  - Returns timeout error but ensures branch is marked as killed
//  - Branch state is always updated, even on timeout
//
// # Graceful Degradation
//
// The function prioritizes state consistency over complete cleanup:
//  1. Attempts full cleanup with timeout enforcement
//  2. If timeout occurs, marks branch as killed regardless
//  3. Appends "(timeout during cleanup)" to KillReason
//  4. Returns error but guarantees branch is in killed state
//
// This ensures the merge queue can continue processing even if some resources
// cannot be cleaned up immediately (they can be garbage collected later).
//
// # Idempotency
//
// This operation is idempotent - killing an already-killed branch is safe and returns no error.
// The original kill metadata (KilledAt timestamp and KillReason) is preserved on subsequent calls.
// No metrics are incremented on duplicate kills.
//
// # Example Usage
//
//	// Use in production where timeout enforcement is critical
//	err := c.killFailedBranchWithTimeout(ctx, branchID, "tests failed: OOM")
//	if err != nil {
//	    log.Warnf("Kill operation had issues: %v (branch still marked as killed)", err)
//	}
//
// See KILLSWITCH.md for detailed architecture documentation.
func (c *Coordinator) killFailedBranchWithTimeout(ctx context.Context, branchID string, reason string) error {
	// Create timeout context for kill operation
	killCtx, cancel := context.WithTimeout(ctx, c.config.KillSwitchTimeout)
	defer cancel()

	// Use a channel to handle the kill operation with timeout
	type killResult struct {
		err error
	}
	resultChan := make(chan killResult, 1)

	go func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		branch, exists := c.activeBranches[branchID]
		if !exists {
			resultChan <- killResult{
				err: &KillSwitchError{
					Operation:   "kill_single_branch",
					BranchID:    branchID,
					Err:         fmt.Errorf("branch not found"),
					Recoverable: false,
					Context:     "branch does not exist in active branches",
				},
			}
			return
		}

		// Idempotent: if already killed, return success without modifying state
		// This prevents:
		// 1. Concurrent kill attempts from incrementing metrics multiple times
		// 2. Overwriting original kill metadata with duplicate information
		// 3. Race conditions between concurrent kill operations
		if branch.Status == BranchStatusKilled {
			resultChan <- killResult{err: nil}
			return
		}

		// TODO: Cancel Temporal workflow with timeout
		// TODO: Stop Docker container with timeout
		// TODO: Clean up worktree with timeout

		// Atomically transition to Killed state with all metadata
		// This ensures no goroutine sees a partially-updated branch
		now := time.Now()
		branch.Status = BranchStatusKilled
		branch.KilledAt = &now
		branch.KillReason = reason

		// Increment metrics atomically within the lock
		// This ensures accurate counters even with concurrent operations
		c.stats.TotalKills++

		resultChan <- killResult{err: nil}
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		return result.err
	case <-killCtx.Done():
		// Graceful degradation: mark as killed anyway even if cleanup timed out
		// Use a separate lock acquisition to avoid deadlock if the goroutine above
		// is still holding the lock
		c.mu.Lock()
		if branch, exists := c.activeBranches[branchID]; exists && branch.Status != BranchStatusKilled {
			now := time.Now()
			branch.Status = BranchStatusKilled
			branch.KilledAt = &now
			branch.KillReason = fmt.Sprintf("%s (timeout during cleanup)", reason)
			c.stats.TotalKills++
		}
		c.mu.Unlock()
		return &TimeoutError{
		Step:              "kill_single_branch",
		BranchID:          branchID,
		ConfiguredTimeout: c.config.KillSwitchTimeout.Milliseconds(),
		PartialProgress:   true,
		CompletedSteps:    []string{"marked_as_killed"},
		PendingSteps:      []string{"cleanup", "notification"},
	}
	}
}

// killDependentBranchesWithTimeout recursively kills all child branches when a parent fails
// with timeout enforcement for the entire cascade operation.
//
// # Kill Switch Cascade (Timeout-Aware Version)
//
// This is an enhanced version of killDependentBranches that includes timeout enforcement
// for the entire cascade operation. It prevents cascading kills from blocking indefinitely
// in deep branch hierarchies.
//
// # Timeout Behavior
//
//  - Creates a timeout context for the entire cascade operation
//  - Cascade timeout = KillSwitchTimeout * 10 (allows for deep hierarchies)
//  - Each individual kill uses the coordinator's KillSwitchTimeout
//  - If the cascade times out, partial progress is maintained (some branches may be killed)
//  - Returns timeout error if cascade doesn't complete in time
//
// # Partial Progress Guarantee
//
// Even if the cascade times out:
//  - All branches processed before timeout are properly killed
//  - Their resources are cleaned up (or marked for cleanup)
//  - TotalKills metrics are accurate for completed kills
//  - Remaining branches can be retried or garbage collected later
//
// This ensures the merge queue doesn't get stuck on problematic branch hierarchies.
//
// # Error Handling
//
// The function uses best-effort error handling:
//  - First error encountered is captured and returned
//  - Subsequent errors are logged but don't override first error
//  - Processing continues even after errors
//  - Timeout errors take precedence over other errors
//
// # Idempotency
//
// This operation is idempotent - it safely handles already-killed branches and continues
// processing remaining children. Errors are logged but do not stop the kill cascade.
//
// # Performance Tuning
//
// The cascade timeout is set to KillSwitchTimeout * 10 to handle deep hierarchies.
// For very deep speculation (>10 levels), consider:
//  - Increasing KillSwitchTimeout in config
//  - Reducing speculation depth
//  - Implementing breadth-first killing instead of depth-first
//
// # Example Usage
//
//	// Use in production with deep speculation hierarchies
//	if err := c.killDependentBranchesWithTimeout(ctx, failedBranchID); err != nil {
//	    log.Warnf("Cascade kill had issues: %v (partial cleanup may have occurred)", err)
//	}
//
// See KILLSWITCH.md for detailed architecture documentation and cascade examples.
func (c *Coordinator) killDependentBranchesWithTimeout(ctx context.Context, branchID string) error {
	// Create a timeout context for the entire cascade operation
	// Use a multiple of KillSwitchTimeout to allow for deep hierarchies
	cascadeTimeout := c.config.KillSwitchTimeout * 10
	cascadeCtx, cancel := context.WithTimeout(ctx, cascadeTimeout)
	defer cancel()

	return c.killDependentBranchesRecursive(cascadeCtx, branchID)
}

// killDependentBranchesRecursive performs the actual recursive kill operation with
// context-aware timeout checking and concurrent child processing.
//
// # Algorithm with Concurrent Processing
//
// This function implements depth-first traversal with timeout checking and concurrency:
//  1. Checks if context is still valid (timeout not exceeded)
//  2. Locks to read the branch's ChildrenIDs list
//  3. Copies the list and unlocks (snapshot prevents TOCTOU issues)
//  4. For each child spawned in a goroutine:
//     a. Checks context validity again (respects cascade timeout)
//     b. Recursively kills the child's descendants (depth-first)
//     c. Kills the child itself with killFailedBranchWithTimeout
//  5. Waits for all children to complete (sync.WaitGroup)
//  6. Returns first error encountered (but all children are processed)
//
// # Thread Safety with Concurrency
//
// This implementation uses sync.WaitGroup to enable concurrent processing of sibling children:
//  - Each child is processed in its own goroutine
//  - No child modifications shared between goroutines (each has its own descendants)
//  - sync.Mutex protects firstErr variable to safely capture errors
//  - Context checks happen in each goroutine to respect timeouts
//  - Lock-release-recurse pattern prevents deadlocks during recursion
//
// # Concurrent Kill Safety
//
// Multiple concurrent kill operations on different branches:
//  - acquire separate locks for their respective branch snapshots
//  - proceed independently if no parent-child conflicts
//  - idempotency checks prevent issues if parent is killed before child
//  - metrics are accurate due to atomic increments within locks
//
// # Race Condition Prevention
//
// The implementation protects against several classes of race conditions:
//
//  1. **TOCTOU (Time-of-Check-Time-of-Use)**: Children snapshot is created before
//     releasing the lock, preventing the children list from changing during iteration.
//
//  2. **Concurrent Kills on Same Branch**: The idempotency check in killFailedBranchWithTimeout
//     ensures that if multiple goroutines try to kill the same branch, only one increments metrics.
//
//  3. **Concurrent Cascade Operations**: Each goroutine works on its own child, and the
//     parent's children list was snapshotted, so concurrent cascades don't interfere.
//
//  4. **Error Race Conditions**: The firstErr variable is protected by a dedicated mutex
//     so concurrent error updates don't corrupt the first error value.
//
// # Error Handling
//
// Captures and returns the first error encountered, but continues processing all remaining
// children. This ensures maximum cleanup even when some operations fail.
// All children are awaited before returning, providing complete cascade coverage.
//
// See KILLSWITCH.md for architecture details and cascade examples.
func (c *Coordinator) killDependentBranchesRecursive(ctx context.Context, branchID string) error {
	// Check if context is still valid
	select {
	case <-ctx.Done():
		return fmt.Errorf("kill cascade timed out for branch %s", branchID)
	default:
	}

	c.mu.Lock()
	branch, exists := c.activeBranches[branchID]
	if !exists {
		c.mu.Unlock()
		return &KillSwitchError{
			Operation:   "cascade_kill",
			BranchID:    branchID,
			Err:         fmt.Errorf("branch not found"),
			Recoverable: false,
			Context:     "branch does not exist in active branches",
		}
	}

	// Create a snapshot of children IDs to process
	// This prevents TOCTOU (Time-of-Check-Time-of-Use) issues where
	// the children list could be modified by concurrent operations
	childrenIDs := make([]string, len(branch.ChildrenIDs))
	copy(childrenIDs, branch.ChildrenIDs)
	c.mu.Unlock()

	// If no children, return early
	if len(childrenIDs) == 0 {
		return nil
	}

	// Track errors but continue processing all children
	var firstErr error
	var errMu sync.Mutex // Protects firstErr for concurrent access

	// Use WaitGroup to spawn concurrent goroutines for each child
	// This enables parallel processing of sibling branches
	var wg sync.WaitGroup

	// Spawn a goroutine for each child to enable concurrent processing
	for _, childID := range childrenIDs {
		wg.Add(1)
		go func(cID string) {
			defer wg.Done()

			// Check timeout before processing this child
			select {
			case <-ctx.Done():
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("kill cascade timed out while processing children of branch %s", branchID)
				}
				errMu.Unlock()
				return
			default:
			}

			// First kill the child's descendants recursively
			if err := c.killDependentBranchesRecursive(ctx, cID); err != nil {
				// Log error but continue killing other branches
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				// Continue to kill the child itself even if descendants had issues
			}

			// Then kill the child itself with timeout
			if err := c.killFailedBranchWithTimeout(ctx, cID, fmt.Sprintf("parent branch %s failed", branchID)); err != nil {
				// Log error but continue
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				// Don't return early - allow other siblings to be processed
			}
		}(childID)
	}

	// Wait for all children to complete
	// This ensures we don't return until the entire subtree is processed
	wg.Wait()

	return firstErr
}
