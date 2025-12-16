// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package workflow provides workflow activity implementations for cell management
package workflow

import (
	"context"
	"fmt"
	"time"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
)

// Activities contains all workflow activities
type Activities struct {
	portManager     infra.PortManagerInterface
	serverManager   infra.ServerManagerInterface
	worktreeManager infra.WorktreeManagerInterface
}

// NewActivities creates a new Activities instance
func NewActivities(portMgr infra.PortManagerInterface, serverMgr infra.ServerManagerInterface, worktreeMgr infra.WorktreeManagerInterface) *Activities {
	return &Activities{
		portManager:     portMgr,
		serverManager:   serverMgr,
		worktreeManager: worktreeMgr,
	}
}

// CellBootstrap represents the resources allocated for an agent cell
type CellBootstrap struct {
	CellID       string
	Port         int
	WorktreeID   string
	WorktreePath string
	ServerHandle *infra.ServerHandle
	Client       agent.ClientInterface
}

// BootstrapCell creates a complete isolated cell for agent execution
// This activity combines port allocation, worktree creation, server boot, and SDK client setup
func (a *Activities) BootstrapCell(ctx context.Context, cellID string, branch string) (*CellBootstrap, error) {
	// 1. Allocate Port (INV-001)
	port, err := a.portManager.Allocate()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// Track port for cleanup on error
	var cleanupPort = true
	defer func() {
		if cleanupPort {
			_ = a.portManager.Release(port)
		}
	}()

	// 2. Create Worktree
	worktreeID := fmt.Sprintf("cell-%s-%d", cellID, time.Now().Unix())
	worktree, err := a.worktreeManager.CreateWorktree(worktreeID, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Track worktree for cleanup on error
	var cleanupWorktree = true
	defer func() {
		if cleanupWorktree {
			_ = a.worktreeManager.RemoveWorktree(worktreeID)
		}
	}()

	// 3. Boot Server (INV-002, INV-003)
	serverHandle, err := a.serverManager.BootServer(ctx, worktree.Path, worktreeID, port)
	if err != nil {
		return nil, fmt.Errorf("failed to boot server: %w", err)
	}

	// Track server for cleanup on error
	var cleanupServer = true
	defer func() {
		if cleanupServer {
			_ = a.serverManager.Shutdown(serverHandle)
		}
	}()

	// 4. Create SDK Client (INV-004)
	client := agent.NewClient(serverHandle.BaseURL, port)

	// Success - disable cleanup
	cleanupPort = false
	cleanupWorktree = false
	cleanupServer = false

	return &CellBootstrap{
		CellID:       cellID,
		Port:         port,
		WorktreeID:   worktreeID,
		WorktreePath: worktree.Path,
		ServerHandle: serverHandle,
		Client:       client,
	}, nil
}

// TeardownCell destroys a cell and releases all resources
// INV-005: Server Process must be killed when Workflow Activity completes
func (a *Activities) TeardownCell(_ context.Context, cell *CellBootstrap) error {
	var errs []error

	// 1. Shutdown Server (INV-005)
	if cell.ServerHandle != nil {
		// Try normal shutdown first (if we have the Cmd object)
		if cell.ServerHandle.Cmd != nil {
			if err := a.serverManager.Shutdown(cell.ServerHandle); err != nil {
				errs = append(errs, fmt.Errorf("failed to shutdown server: %w", err))
			}
		} else if cell.ServerHandle.PID != 0 {
			// Fallback to PID-based shutdown (after serialization)
			if err := a.serverManager.ShutdownByPID(cell.ServerHandle.PID); err != nil {
				errs = append(errs, fmt.Errorf("failed to shutdown server by PID: %w", err))
			}
		}
	}

	// 2. Remove Worktree
	if cell.WorktreeID != "" {
		if err := a.worktreeManager.RemoveWorktree(cell.WorktreeID); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove worktree: %w", err))
		}
	}

	// 3. Release Port
	if cell.Port != 0 {
		if err := a.portManager.Release(cell.Port); err != nil {
			errs = append(errs, fmt.Errorf("failed to release port: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}

	return nil
}

// ExecuteTask runs a task within a cell
// INV-006: Command execution must use SDK
func (a *Activities) ExecuteTask(ctx context.Context, cell *CellBootstrap, task *agent.TaskContext) (*agent.ExecutionResult, error) {
	// 1. Verify server is healthy
	if !a.serverManager.IsHealthy(ctx, cell.ServerHandle) {
		return nil, fmt.Errorf("server is not healthy")
	}

	// 2. Execute prompt via SDK
	result, err := cell.Client.ExecutePrompt(ctx, task.Prompt, &agent.PromptOptions{
		Title: fmt.Sprintf("Task: %s", task.TaskID),
		Agent: "build", // Use build agent for code changes
	})
	if err != nil {
		return &agent.ExecutionResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	// 3. Get file modifications
	fileStatus, err := cell.Client.GetFileStatus(ctx)
	if err != nil {
		return &agent.ExecutionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get file status: %v", err),
		}, nil
	}

	filesModified := make([]string, 0)
	for _, file := range fileStatus {
		// File.Path is the file path, check if it exists (modified/added files)
		if file.Path != "" {
			filesModified = append(filesModified, file.Path)
		}
	}

	// 4. Construct result
	executionResult := &agent.ExecutionResult{
		Success:       true,
		Output:        result.GetText(),
		FilesModified: filesModified,
		SessionID:     result.SessionID,
	}

	return executionResult, nil
}

// RunTests executes tests in the cell
func (a *Activities) RunTests(ctx context.Context, cell *CellBootstrap) (bool, error) {
	// Execute test command via prompt (creates session automatically)
	result, err := cell.Client.ExecutePrompt(ctx, "Run: go test ./...", &agent.PromptOptions{
		Title: "Run Tests",
		Agent: "build",
	})
	if err != nil {
		return false, fmt.Errorf("failed to execute tests: %w", err)
	}

	// Check if tests passed (very basic check)
	output := result.GetText()
	testsPassed := !containsString(output, "FAIL")

	return testsPassed, nil
}

// CommitChanges commits changes in the worktree
func (a *Activities) CommitChanges(ctx context.Context, cell *CellBootstrap, message string) error {
	// Execute git commands via prompt
	prompt := fmt.Sprintf("Run these git commands:\ngit add .\ngit commit -m \"%s\"", message)
	_, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: "Commit Changes",
		Agent: "build",
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// RevertChanges reverts all changes in the worktree
func (a *Activities) RevertChanges(ctx context.Context, cell *CellBootstrap) error {
	// Execute git reset via prompt
	_, err := cell.Client.ExecutePrompt(ctx, "Run: git reset --hard", &agent.PromptOptions{
		Title: "Revert Changes",
		Agent: "build",
	})
	if err != nil {
		return fmt.Errorf("failed to revert changes: %w", err)
	}

	return nil
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			s[1:len(substr)+1] == substr))
}
