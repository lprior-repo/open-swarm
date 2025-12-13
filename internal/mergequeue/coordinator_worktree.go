package mergequeue

import (
	"context"
	"path/filepath"

	"open-swarm/internal/temporal"
)

// cleanupBranchWorktrees removes all worktrees associated with a killed branch.
// This ensures that git worktrees are properly cleaned up when branches are killed,
// preventing orphaned worktrees from accumulating on disk.
func (c *Coordinator) cleanupBranchWorktrees(ctx context.Context, branch *SpeculativeBranch) error {
	// Get the global worktree manager from temporal package
	_, _, worktreeManager := temporal.GetManagers()
	if worktreeManager == nil {
		// Worktree manager not initialized, skip cleanup
		return nil
	}

	var lastErr error
	for _, change := range branch.Changes {
		if change.WorktreePath == "" {
			continue
		}

		// Extract worktree ID from the path
		// The worktree path typically follows the pattern: /base/dir/worktreeID
		worktreeID := filepath.Base(change.WorktreePath)
		if worktreeID == "" || worktreeID == "." || worktreeID == "/" {
			continue
		}

		// Remove the worktree
		if err := worktreeManager.RemoveWorktree(worktreeID); err != nil {
			// Log error but continue with other worktrees
			// The worktree may have already been removed or may not exist
			lastErr = err
			continue
		}
	}

	return lastErr
}
