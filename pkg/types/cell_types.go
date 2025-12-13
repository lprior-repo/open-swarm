// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package types

// ============================================================================
// CELL LIFECYCLE TYPES
// ============================================================================

// BootstrapInput defines input parameters for cell bootstrap operations.
// It specifies which cell to create and which git branch to work on.
type BootstrapInput struct {
	// CellID is the unique identifier for the cell to bootstrap
	CellID string

	// Branch is the git branch to checkout in the cell's worktree
	Branch string
}

// BootstrapOutput contains the serializable results of cell bootstrap.
// This includes all information needed to interact with the bootstrapped cell,
// such as port numbers, paths, and process identifiers.
//
// Note: This is a serializable type safe for Temporal workflow state.
// Runtime handles (like process objects) are not included.
type BootstrapOutput struct {
	// CellID is the unique identifier for this cell
	CellID string

	// Port is the network port the cell server is listening on
	Port int

	// WorktreeID is the identifier for the git worktree
	WorktreeID string

	// WorktreePath is the absolute filesystem path to the worktree
	WorktreePath string

	// BaseURL is the base URL for the cell server (e.g., http://localhost:PORT)
	BaseURL string

	// ServerPID is the process ID of the running cell server
	ServerPID int
}

// ============================================================================
// TASK EXECUTION TYPES
// ============================================================================

// TaskInput defines input parameters for task execution within a cell.
// It contains all information needed to execute a single task.
type TaskInput struct {
	// TaskID is the unique identifier for this task
	TaskID string

	// Description is a human-readable description of what the task does
	Description string

	// Prompt is the instruction given to the agent for task execution
	Prompt string
}

// TaskOutput contains the results of task execution within a cell.
// It includes success status, output, and any files that were modified.
type TaskOutput struct {
	// Success indicates whether the task completed successfully
	Success bool

	// Output contains the console output or result from task execution
	Output string

	// FilesModified is the list of files changed during task execution
	FilesModified []string

	// ErrorMessage contains any error message if the task failed
	ErrorMessage string
}
