// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager manages Git worktrees for agent isolation
type WorktreeManager struct {
	baseDir string
	repoDir string
}

// WorktreeInfo contains information about a worktree
type WorktreeInfo struct {
	ID   string
	Path string
}

// NewWorktreeManager creates a new worktree manager
func NewWorktreeManager(repoDir, baseDir string) *WorktreeManager {
	return &WorktreeManager{
		repoDir: repoDir,
		baseDir: baseDir,
	}
}

// CreateWorktree creates a new Git worktree for agent isolation
func (wm *WorktreeManager) CreateWorktree(id string, branch string) (*WorktreeInfo, error) {
	worktreePath := filepath.Join(wm.baseDir, id)

	// Ensure base directory exists
	if err := os.MkdirAll(wm.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return nil, fmt.Errorf("worktree %s already exists at %s", id, worktreePath)
	}

	// Create worktree: git worktree add <path> <branch>
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Dir = wm.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	return &WorktreeInfo{
		ID:   id,
		Path: worktreePath,
	}, nil
}

// RemoveWorktree removes a Git worktree
func (wm *WorktreeManager) RemoveWorktree(id string) error {
	worktreePath := filepath.Join(wm.baseDir, id)

	// Remove worktree: git worktree remove <path>
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = wm.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ListWorktrees lists all worktrees in the repository
func (wm *WorktreeManager) ListWorktrees() ([]*WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = wm.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return wm.parseWorktreeList(string(output))
}

// parseWorktreeList parses the output of git worktree list --porcelain
func (wm *WorktreeManager) parseWorktreeList(output string) ([]*WorktreeInfo, error) {
	var worktrees []*WorktreeInfo
	lines := strings.Split(output, "\n")

	var currentPath string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if currentPath != "" && strings.HasPrefix(currentPath, wm.baseDir) {
			// Extract ID from path
			id := filepath.Base(currentPath)
			worktrees = append(worktrees, &WorktreeInfo{
				ID:   id,
				Path: currentPath,
			})
			currentPath = ""
		}
	}

	return worktrees, nil
}

// PruneWorktrees removes worktree administrative information for missing worktrees
func (wm *WorktreeManager) PruneWorktrees() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = wm.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// CleanupAll removes all worktrees in the base directory
func (wm *WorktreeManager) CleanupAll() error {
	worktrees, err := wm.ListWorktrees()
	if err != nil {
		return err
	}

	var errs []error
	for _, wt := range worktrees {
		if err := wm.RemoveWorktree(wt.ID); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to cleanup some worktrees: %v", errs)
	}

	return wm.PruneWorktrees()
}
