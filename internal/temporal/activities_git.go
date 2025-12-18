// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

// gitSafeGetLogger returns the activity logger if in activity context, otherwise a noop logger
func gitSafeGetLogger(ctx context.Context) (logger log.Logger) {
	defer func() {
		if r := recover(); r != nil {
			// Not in an activity context, return noop logger
			logger = noopLogger{}
		}
	}()
	logger = activity.GetLogger(ctx)
	return logger
}

// gitSafeRecordHeartbeat records a heartbeat if in activity context
func gitSafeRecordHeartbeat(ctx context.Context, details ...interface{}) {
	// Only record heartbeat if we're in an activity context
	// This will panic if not, so we recover
	defer func() {
		recover()
	}()
	activity.RecordHeartbeat(ctx, details...)
}

// GitActivities provides thin wrappers around Git operations
// These activities are designed to be:
// - Idempotent where possible
// - Minimal business logic
// - Proper error handling
// - Structured results
type GitActivities struct{}

// GitCloneInput specifies parameters for cloning a repository
type GitCloneInput struct {
	URL       string // Repository URL to clone
	TargetDir string // Directory to clone into
	Branch    string // Optional: specific branch to checkout (empty = default branch)
	Depth     int    // Optional: shallow clone depth (0 = full clone)
}

// GitCloneOutput contains the result of a clone operation
type GitCloneOutput struct {
	Path   string // Path to the cloned repository
	Branch string // Current branch after clone
	Commit string // Current commit hash
}

// GitClone clones a Git repository to the specified directory
// Idempotent: If directory exists and is a valid git repo with same remote, returns success
func (ga *GitActivities) GitClone(ctx context.Context, input GitCloneInput) (*GitCloneOutput, error) {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Cloning Git repository", "url", input.URL, "target", input.TargetDir)

	gitSafeRecordHeartbeat(ctx, "executing")

	// Build git clone command
	args := []string{"clone", input.URL, input.TargetDir}
	if input.Branch != "" {
		args = append(args, "-b", input.Branch)
	}
	if input.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", input.Depth))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already exists with same remote (idempotent)
		if strings.Contains(string(output), "already exists") {
			// Verify it's the same repository
			remoteCmd := exec.CommandContext(ctx, "git", "-C", input.TargetDir, "remote", "get-url", "origin")
			remoteURL, remoteErr := remoteCmd.Output()
			if remoteErr == nil && strings.TrimSpace(string(remoteURL)) == input.URL {
				logger.Info("Repository already cloned with matching remote")
				return ga.getRepoInfo(ctx, input.TargetDir)
			}
		}
		return nil, fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	logger.Info("Repository cloned successfully", "path", input.TargetDir)
	return ga.getRepoInfo(ctx, input.TargetDir)
}

// GitCheckoutInput specifies parameters for checking out a branch
type GitCheckoutInput struct {
	RepoPath   string // Path to the repository
	Branch     string // Branch name to checkout
	CreateNew  bool   // If true, create new branch
	StartPoint string // Optional: starting point for new branch (commit/branch)
}

// GitCheckout checks out a branch (or creates and checks out a new branch)
// Idempotent: If already on the specified branch, returns success
func (ga *GitActivities) GitCheckout(ctx context.Context, input GitCheckoutInput) error {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Checking out branch", "repo", input.RepoPath, "branch", input.Branch, "create", input.CreateNew)

	gitSafeRecordHeartbeat(ctx, "executing")

	// Check current branch (idempotent check)
	currentBranchCmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "branch", "--show-current")
	currentBranch, _ := currentBranchCmd.Output()
	if strings.TrimSpace(string(currentBranch)) == input.Branch {
		logger.Info("Already on target branch", "branch", input.Branch)
		return nil
	}

	args := []string{"-C", input.RepoPath, "checkout"}
	if input.CreateNew {
		args = append(args, "-b")
	}
	args = append(args, input.Branch)
	if input.StartPoint != "" {
		args = append(args, input.StartPoint)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}

	logger.Info("Branch checked out successfully", "branch", input.Branch)
	return nil
}

// GitCommitInput specifies parameters for creating a commit
type GitCommitInput struct {
	RepoPath string   // Path to the repository
	Message  string   // Commit message
	Files    []string // Files to stage (empty = all changes)
	Author   string   // Optional: author in "Name <email>" format
}

// GitCommitOutput contains the result of a commit operation
type GitCommitOutput struct {
	CommitHash string // Hash of the created commit
	Files      int    // Number of files changed
}

// GitCommit creates a Git commit with the specified files and message
// Not idempotent by nature, but safely handles "nothing to commit" case
func (ga *GitActivities) GitCommit(ctx context.Context, input GitCommitInput) (*GitCommitOutput, error) {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Creating Git commit", "repo", input.RepoPath, "message", input.Message)

	gitSafeRecordHeartbeat(ctx, "executing")

	// Stage files
	if len(input.Files) == 0 {
		// Add all changes
		addCmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "add", ".")
		if output, err := addCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git add failed: %w\nOutput: %s", err, string(output))
		}
	} else {
		// Add specific files
		args := append([]string{"-C", input.RepoPath, "add"}, input.Files...)
		addCmd := exec.CommandContext(ctx, "git", args...)
		if output, err := addCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git add failed: %w\nOutput: %s", err, string(output))
		}
	}

	// Check if there are changes to commit
	statusCmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "status", "--porcelain")
	statusOutput, _ := statusCmd.Output()
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		logger.Info("No changes to commit")
		return &GitCommitOutput{CommitHash: "", Files: 0}, nil
	}

	// Create commit
	commitArgs := []string{"-C", input.RepoPath, "commit", "-m", input.Message}
	if input.Author != "" {
		commitArgs = append(commitArgs, "--author", input.Author)
	}

	commitCmd := exec.CommandContext(ctx, "git", commitArgs...)
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	// Get commit hash
	hashCmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "rev-parse", "HEAD")
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Count files changed
	filesCmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "diff", "--name-only", "HEAD~1", "HEAD")
	filesOutput, _ := filesCmd.Output()
	fileCount := len(strings.Split(strings.TrimSpace(string(filesOutput)), "\n"))

	result := &GitCommitOutput{
		CommitHash: strings.TrimSpace(string(hashOutput)),
		Files:      fileCount,
	}

	logger.Info("Commit created successfully", "hash", result.CommitHash, "files", result.Files)
	return result, nil
}

// GitPushInput specifies parameters for pushing to remote
type GitPushInput struct {
	RepoPath    string // Path to the repository
	Remote      string // Remote name (typically "origin")
	Branch      string // Branch to push
	Force       bool   // Whether to force push
	SetUpstream bool   // Whether to set upstream tracking
}

// GitPush pushes commits to a remote repository
// Idempotent: If remote is already up-to-date, returns success
func (ga *GitActivities) GitPush(ctx context.Context, input GitPushInput) error {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Pushing to remote", "repo", input.RepoPath, "remote", input.Remote, "branch", input.Branch)

	gitSafeRecordHeartbeat(ctx, "executing")

	args := []string{"-C", input.RepoPath, "push"}
	if input.Force {
		args = append(args, "--force")
	}
	if input.SetUpstream {
		args = append(args, "--set-upstream")
	}
	args = append(args, input.Remote, input.Branch)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already up-to-date (idempotent)
		if strings.Contains(string(output), "up-to-date") || strings.Contains(string(output), "up to date") {
			logger.Info("Remote already up-to-date")
			return nil
		}
		return fmt.Errorf("git push failed: %w\nOutput: %s", err, string(output))
	}

	logger.Info("Pushed successfully to remote")
	return nil
}

// GitStatusOutput contains the status of the repository
type GitStatusOutput struct {
	Branch    string   // Current branch
	Modified  []string // Modified files
	Untracked []string // Untracked files
	Staged    []string // Staged files
	IsDirty   bool     // Whether there are uncommitted changes
	AheadBy   int      // Commits ahead of remote
	BehindBy  int      // Commits behind remote
}

// GitStatus returns the current status of the repository
//
//nolint:cyclop // complexity 13 is acceptable for status parsing
func (ga *GitActivities) GitStatus(ctx context.Context, repoPath string) (*GitStatusOutput, error) {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Getting Git status", "repo", repoPath)

	result := &GitStatusOutput{
		Modified:  []string{},
		Untracked: []string{},
		Staged:    []string{},
	}

	// Get current branch
	branchCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	result.Branch = strings.TrimSpace(string(branchOutput))

	// Get status in porcelain format
	statusCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// Parse status output
	lines := strings.Split(string(statusOutput), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])

		if status[0] != ' ' && status[0] != '?' {
			result.Staged = append(result.Staged, file)
		}
		if status[1] == 'M' {
			result.Modified = append(result.Modified, file)
		}
		if status == "??" {
			result.Untracked = append(result.Untracked, file)
		}
	}

	result.IsDirty = len(result.Modified) > 0 || len(result.Untracked) > 0 || len(result.Staged) > 0

	// Get ahead/behind count
	revListCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	revListOutput, err := revListCmd.Output()
	if err == nil {
		parts := strings.Fields(string(revListOutput))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &result.AheadBy)
			fmt.Sscanf(parts[1], "%d", &result.BehindBy)
		}
	}

	logger.Info("Status retrieved", "branch", result.Branch, "dirty", result.IsDirty)
	return result, nil
}

// GitDiffOutput contains diff information
type GitDiffOutput struct {
	Diff         string   // Full diff text
	FilesChanged []string // List of changed files
	Insertions   int      // Number of insertions
	Deletions    int      // Number of deletions
}

// GitDiff returns the diff between two commits/branches
func (ga *GitActivities) GitDiff(ctx context.Context, repoPath, from, to string) (*GitDiffOutput, error) {
	logger := gitSafeGetLogger(ctx)
	logger.Info("Getting Git diff", "repo", repoPath, "from", from, "to", to)

	// Get diff
	diffCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", from, to)
	diffOutput, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	// Get file list
	filesCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", from, to)
	filesOutput, err := filesCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	files := []string{}
	for _, line := range strings.Split(string(filesOutput), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}

	// Get stats
	statsCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--stat", from, to)
	statsOutput, _ := statsCmd.Output()

	result := &GitDiffOutput{
		Diff:         string(diffOutput),
		FilesChanged: files,
		Insertions:   0,
		Deletions:    0,
	}

	// Parse stats for insertions/deletions
	statsLines := strings.Split(string(statsOutput), "\n")
	if len(statsLines) > 0 {
		lastLine := statsLines[len(statsLines)-1]
		fmt.Sscanf(lastLine, " %d files changed, %d insertions(+), %d deletions(-)",
			&result.Insertions, &result.Deletions, &result.Deletions)
	}

	logger.Info("Diff retrieved", "files", len(files))
	return result, nil
}

// GitBranchInput specifies parameters for branch operations
type GitBranchInput struct {
	RepoPath string // Path to the repository
	Name     string // Branch name
	Delete   bool   // Whether to delete the branch
	Force    bool   // Force delete (for -D flag)
}

// GitBranch creates or deletes a branch
// For creation: Idempotent - if branch exists, returns success
// For deletion: Idempotent - if branch doesn't exist, returns success
func (ga *GitActivities) GitBranch(ctx context.Context, input GitBranchInput) error {
	logger := gitSafeGetLogger(ctx)

	if input.Delete {
		logger.Info("Deleting branch", "repo", input.RepoPath, "branch", input.Name)

		args := []string{"-C", input.RepoPath, "branch"}
		if input.Force {
			args = append(args, "-D")
		} else {
			args = append(args, "-d")
		}
		args = append(args, input.Name)

		cmd := exec.CommandContext(ctx, "git", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Branch not found is OK (idempotent)
			if strings.Contains(string(output), "not found") {
				logger.Info("Branch already deleted")
				return nil
			}
			return fmt.Errorf("git branch delete failed: %w\nOutput: %s", err, string(output))
		}
		logger.Info("Branch deleted successfully")
	} else {
		logger.Info("Creating branch", "repo", input.RepoPath, "branch", input.Name)

		cmd := exec.CommandContext(ctx, "git", "-C", input.RepoPath, "branch", input.Name)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Branch exists is OK (idempotent)
			if strings.Contains(string(output), "already exists") {
				logger.Info("Branch already exists")
				return nil
			}
			return fmt.Errorf("git branch create failed: %w\nOutput: %s", err, string(output))
		}
		logger.Info("Branch created successfully")
	}

	return nil
}

// getRepoInfo is a helper to extract repository information
func (ga *GitActivities) getRepoInfo(ctx context.Context, repoPath string) (*GitCloneOutput, error) {
	// Get current branch
	branchCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	// Get current commit
	commitCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	commitOutput, err := commitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &GitCloneOutput{
		Path:   repoPath,
		Branch: strings.TrimSpace(string(branchOutput)),
		Commit: strings.TrimSpace(string(commitOutput)),
	}, nil
}
