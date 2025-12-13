// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"context"
)

// PortManagerInterface defines the interface for port allocation management
type PortManagerInterface interface {
	// Allocate reserves the next available port
	Allocate() (int, error)

	// Release frees a previously allocated port
	Release(port int) error

	// AllocatedCount returns the number of currently allocated ports
	AllocatedCount() int

	// IsAllocated checks if a specific port is currently allocated
	IsAllocated(port int) bool

	// AvailableCount returns the number of available ports
	AvailableCount() int
}

// ServerManagerInterface defines the interface for server lifecycle management
type ServerManagerInterface interface {
	// BootServer starts an opencode server on the specified port and working directory
	BootServer(ctx context.Context, worktreePath string, worktreeID string, port int) (*ServerHandle, error)

	// Shutdown gracefully stops the opencode server
	Shutdown(handle *ServerHandle) error

	// IsHealthy checks if the server is still responsive
	IsHealthy(ctx context.Context, handle *ServerHandle) bool
}

// WorktreeManagerInterface defines the interface for Git worktree management
type WorktreeManagerInterface interface {
	// CreateWorktree creates a new Git worktree for agent isolation
	CreateWorktree(id string, branch string) (*WorktreeInfo, error)

	// RemoveWorktree removes a Git worktree
	RemoveWorktree(id string) error

	// ListWorktrees lists all worktrees in the repository
	ListWorktrees() ([]*WorktreeInfo, error)

	// PruneWorktrees removes worktree administrative information for missing worktrees
	PruneWorktrees() error

	// CleanupAll removes all worktrees in the base directory
	CleanupAll() error
}

// Ensure concrete types implement interfaces
var _ PortManagerInterface = (*PortManager)(nil)
var _ ServerManagerInterface = (*ServerManager)(nil)
var _ WorktreeManagerInterface = (*WorktreeManager)(nil)
