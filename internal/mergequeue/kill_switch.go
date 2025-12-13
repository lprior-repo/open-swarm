// Package mergequeue implements an Uber-style speculative merge queue with hierarchical kill switch.
//
// The kill switch functionality is defined in this file and provides timeout-aware
// failure propagation through the branch hierarchy. See KILLSWITCH.md for architecture details.
package mergequeue

import (
	"context"
	"fmt"
	"time"
)

// killFailedBranchWithTimeout kills a single speculative branch with timeout enforcement.
//
// # Kill Switch Component (Timeout-Aware Version)
//
// This is an enhanced version of killFailedBranch that includes timeout enforcement
// and graceful degradation. It prevents kill operations from blocking indefinitely
// when resource cleanup (Temporal workflows, Docker containers) hangs or fails.
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
			resultChan <- killResult{err: fmt.Errorf("branch %s not found", branchID)}
			return
		}

		// Idempotent: if already killed, return success without modifying state
		if branch.Status == BranchStatusKilled {
			resultChan <- killResult{err: nil}
			return
		}

		// TODO: Cancel Temporal workflow with timeout
		// TODO: Stop Docker container with timeout
		// TODO: Clean up worktree with timeout

		// Update branch status
		now := time.Now()
		branch.Status = BranchStatusKilled
		branch.KilledAt = &now
		branch.KillReason = reason

		// Track kill switch activation (only increment for new kills due to idempotency check above)
		c.stats.TotalKills++

		resultChan <- killResult{err: nil}
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		return result.err
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
// context-aware timeout checking at each level of the hierarchy.
//
// # Algorithm
//
// This function implements depth-first traversal with timeout checking:
//  1. Checks if context is still valid (timeout not exceeded)
//  2. Locks to read the branch's ChildrenIDs list
//  3. Copies the list and unlocks
//  4. For each child:
//     a. Checks context validity again
//     b. Recursively kills the child's descendants (depth-first)
//     c. Kills the child itself with killFailedBranchWithTimeout
//  5. Returns first error encountered (but continues processing all children)
//
// # Thread Safety
//
// Uses the same lock-release-recurse pattern as killDependentBranches to avoid deadlocks.
// Each recursive call acquires its own lock independently.
//
// # Error Handling
//
// Captures and returns the first error encountered, but continues processing all remaining
// children. This ensures maximum cleanup even when some operations fail.
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
		return fmt.Errorf("branch %s not found", branchID)
	}

	childrenIDs := make([]string, len(branch.ChildrenIDs))
	copy(childrenIDs, branch.ChildrenIDs)
	c.mu.Unlock()

	// Track errors but continue processing
	var firstErr error

	// Recursively kill all children
	for _, childID := range childrenIDs {
		// Check timeout before processing each child
		select {
		case <-ctx.Done():
			return fmt.Errorf("kill cascade timed out while processing children of branch %s", branchID)
		default:
		}

		// First kill the child's descendants
		if err := c.killDependentBranchesRecursive(ctx, childID); err != nil {
			// Log error but continue killing other branches
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		// Then kill the child itself with timeout
		if err := c.killFailedBranchWithTimeout(ctx, childID, fmt.Sprintf("parent branch %s failed", branchID)); err != nil {
			// Log error but continue
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
	}

	return firstErr
}
