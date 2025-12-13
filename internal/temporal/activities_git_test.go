// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary Git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run(), "failed to init repo")

	// Configure user
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	require.NoError(t, os.WriteFile(testFile, []byte("# Test Repo"), 0644))

	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit").Run()

	return tmpDir
}

func TestGitActivities_GitStatus(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	t.Run("clean repository", func(t *testing.T) {
		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.False(t, status.IsDirty)
		assert.Empty(t, status.Modified)
		assert.Empty(t, status.Untracked)
		assert.NotEmpty(t, status.Branch)
	})

	t.Run("modified file", func(t *testing.T) {
		testFile := filepath.Join(repoPath, "README.md")
		require.NoError(t, os.WriteFile(testFile, []byte("# Modified"), 0644))

		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.True(t, status.IsDirty)
		assert.Contains(t, status.Modified, "README.md")
	})

	t.Run("untracked file", func(t *testing.T) {
		newFile := filepath.Join(repoPath, "new.txt")
		require.NoError(t, os.WriteFile(newFile, []byte("new"), 0644))

		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.True(t, status.IsDirty)
		assert.Contains(t, status.Untracked, "new.txt")
	})
}

func TestGitActivities_GitCheckout(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	t.Run("create new branch", func(t *testing.T) {
		input := GitCheckoutInput{
			RepoPath:  repoPath,
			Branch:    "feature-test",
			CreateNew: true,
		}

		err := ga.GitCheckout(ctx, input)
		require.NoError(t, err)

		// Verify we're on the new branch
		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, "feature-test", status.Branch)
	})

	t.Run("idempotent - already on branch", func(t *testing.T) {
		input := GitCheckoutInput{
			RepoPath:  repoPath,
			Branch:    "feature-test",
			CreateNew: false,
		}

		// Should succeed without error even though already on this branch
		err := ga.GitCheckout(ctx, input)
		require.NoError(t, err)
	})

	t.Run("switch to existing branch", func(t *testing.T) {
		// Create another branch
		exec.Command("git", "-C", repoPath, "checkout", "-b", "another-branch").Run()

		input := GitCheckoutInput{
			RepoPath:  repoPath,
			Branch:    "feature-test",
			CreateNew: false,
		}

		err := ga.GitCheckout(ctx, input)
		require.NoError(t, err)

		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, "feature-test", status.Branch)
	})
}

func TestGitActivities_GitCommit(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	t.Run("commit with changes", func(t *testing.T) {
		// Create a new file
		testFile := filepath.Join(repoPath, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

		input := GitCommitInput{
			RepoPath: repoPath,
			Message:  "Test commit",
			Files:    []string{},
		}

		output, err := ga.GitCommit(ctx, input)
		require.NoError(t, err)
		assert.NotEmpty(t, output.CommitHash)
		assert.Greater(t, output.Files, 0)
	})

	t.Run("no changes to commit", func(t *testing.T) {
		input := GitCommitInput{
			RepoPath: repoPath,
			Message:  "No changes",
			Files:    []string{},
		}

		output, err := ga.GitCommit(ctx, input)
		require.NoError(t, err)
		assert.Empty(t, output.CommitHash)
		assert.Equal(t, 0, output.Files)
	})

	t.Run("commit specific files", func(t *testing.T) {
		// Create multiple files
		file1 := filepath.Join(repoPath, "file1.txt")
		file2 := filepath.Join(repoPath, "file2.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0644))

		input := GitCommitInput{
			RepoPath: repoPath,
			Message:  "Commit specific file",
			Files:    []string{"file1.txt"},
		}

		output, err := ga.GitCommit(ctx, input)
		require.NoError(t, err)
		assert.NotEmpty(t, output.CommitHash)

		// Verify only file1 was committed
		status, err := ga.GitStatus(ctx, repoPath)
		require.NoError(t, err)
		assert.Contains(t, status.Untracked, "file2.txt")
	})

	t.Run("commit with custom author", func(t *testing.T) {
		testFile := filepath.Join(repoPath, "author-test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		input := GitCommitInput{
			RepoPath: repoPath,
			Message:  "Custom author commit",
			Files:    []string{},
			Author:   "Custom Author <custom@example.com>",
		}

		output, err := ga.GitCommit(ctx, input)
		require.NoError(t, err)
		assert.NotEmpty(t, output.CommitHash)
	})
}

func TestGitActivities_GitBranch(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	t.Run("create branch", func(t *testing.T) {
		input := GitBranchInput{
			RepoPath: repoPath,
			Name:     "new-branch",
			Delete:   false,
		}

		err := ga.GitBranch(ctx, input)
		require.NoError(t, err)

		// Verify branch exists
		cmd := exec.Command("git", "-C", repoPath, "branch", "--list", "new-branch")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "new-branch")
	})

	t.Run("idempotent - branch already exists", func(t *testing.T) {
		input := GitBranchInput{
			RepoPath: repoPath,
			Name:     "new-branch",
			Delete:   false,
		}

		// Should succeed even though branch exists
		err := ga.GitBranch(ctx, input)
		require.NoError(t, err)
	})

	t.Run("delete branch", func(t *testing.T) {
		// Create a branch to delete
		exec.Command("git", "-C", repoPath, "branch", "to-delete").Run()

		input := GitBranchInput{
			RepoPath: repoPath,
			Name:     "to-delete",
			Delete:   true,
		}

		err := ga.GitBranch(ctx, input)
		require.NoError(t, err)

		// Verify branch is gone
		cmd := exec.Command("git", "-C", repoPath, "branch", "--list", "to-delete")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.NotContains(t, string(output), "to-delete")
	})

	t.Run("idempotent - delete non-existent branch", func(t *testing.T) {
		input := GitBranchInput{
			RepoPath: repoPath,
			Name:     "non-existent",
			Delete:   true,
		}

		// Should succeed even though branch doesn't exist
		err := ga.GitBranch(ctx, input)
		require.NoError(t, err)
	})
}

func TestGitActivities_GitDiff(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	// Create a second commit
	testFile := filepath.Join(repoPath, "diff-test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("version 1"), 0644))
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Second commit").Run()

	// Get commit hashes
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD~1")
	firstCommit, err := cmd.Output()
	require.NoError(t, err)

	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	secondCommit, err := cmd.Output()
	require.NoError(t, err)

	t.Run("diff between commits", func(t *testing.T) {
		diff, err := ga.GitDiff(ctx, repoPath,
			strings.TrimSpace(string(firstCommit)),
			strings.TrimSpace(string(secondCommit)))
		require.NoError(t, err)
		assert.NotEmpty(t, diff.Diff)
		assert.Contains(t, diff.FilesChanged, "diff-test.txt")
	})

	t.Run("no diff - same commit", func(t *testing.T) {
		commit := strings.TrimSpace(string(secondCommit))
		diff, err := ga.GitDiff(ctx, repoPath, commit, commit)
		require.NoError(t, err)
		assert.Empty(t, diff.Diff)
		assert.Empty(t, diff.FilesChanged)
	})
}

func TestGitActivities_GitClone(t *testing.T) {
	// Create a source repo to clone from
	sourceRepo := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	targetDir := t.TempDir()
	cloneDir := filepath.Join(targetDir, "cloned-repo")

	t.Run("clone repository", func(t *testing.T) {
		input := GitCloneInput{
			URL:       sourceRepo,
			TargetDir: cloneDir,
		}

		output, err := ga.GitClone(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, cloneDir, output.Path)
		assert.NotEmpty(t, output.Branch)
		assert.NotEmpty(t, output.Commit)

		// Verify repo exists
		_, err = os.Stat(filepath.Join(cloneDir, ".git"))
		require.NoError(t, err)
	})

	t.Run("idempotent - already cloned with same remote", func(t *testing.T) {
		input := GitCloneInput{
			URL:       sourceRepo,
			TargetDir: cloneDir,
		}

		// Should succeed even though already cloned
		output, err := ga.GitClone(ctx, input)
		require.NoError(t, err)
		assert.NotEmpty(t, output.Path)
	})

	t.Run("clone with specific branch", func(t *testing.T) {
		// Create a branch in source
		exec.Command("git", "-C", sourceRepo, "checkout", "-b", "test-branch").Run()

		cloneDir2 := filepath.Join(targetDir, "cloned-branch")
		input := GitCloneInput{
			URL:       sourceRepo,
			TargetDir: cloneDir2,
			Branch:    "test-branch",
		}

		output, err := ga.GitClone(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, "test-branch", output.Branch)
	})

	t.Run("shallow clone", func(t *testing.T) {
		cloneDir3 := filepath.Join(targetDir, "shallow-clone")
		input := GitCloneInput{
			URL:       sourceRepo,
			TargetDir: cloneDir3,
			Depth:     1,
		}

		output, err := ga.GitClone(ctx, input)
		require.NoError(t, err)
		assert.NotEmpty(t, output.Path)
	})
}

func TestGitActivities_GitPush(t *testing.T) {
	// Note: This test requires a remote setup, which is complex in unit tests
	// In practice, you would use a bare repository as a remote
	// Here we'll just test the basic error cases

	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	t.Run("push without remote", func(t *testing.T) {
		input := GitPushInput{
			RepoPath: repoPath,
			Remote:   "origin",
			Branch:   "main",
		}

		err := ga.GitPush(ctx, input)
		// Should fail because there's no remote configured
		assert.Error(t, err)
	})
}

func TestGitActivities_getRepoInfo(t *testing.T) {
	repoPath := setupTestRepo(t)
	ga := &GitActivities{}
	ctx := context.Background()

	info, err := ga.getRepoInfo(ctx, repoPath)
	require.NoError(t, err)
	assert.NotEmpty(t, info.Branch)
	assert.NotEmpty(t, info.Commit)
	assert.Equal(t, repoPath, info.Path)
}
