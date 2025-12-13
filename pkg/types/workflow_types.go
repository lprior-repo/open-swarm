// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package types provides shared workflow types used across Open Swarm.
//
// This package contains core workflow types that are shared between different
// packages to break circular dependencies. Types here should be:
// - Pure data structures (no behavior)
// - Serializable for Temporal workflows
// - Stable and version-controlled
//
// Design principles:
// - Domain-centric: Types represent workflow domain concepts
// - Dependency-free: No imports from internal packages
// - Composable: Types can be combined and extended
package types

// ============================================================================
// DAG WORKFLOW TYPES
// ============================================================================

// Task represents a single task in a DAG workflow.
// Each task has a name, command to execute, and zero or more dependencies
// that must complete before this task can run.
type Task struct {
	// Name is the unique identifier for this task within the DAG
	Name string

	// Command is the shell command to execute for this task
	Command string

	// Deps is the list of task names that must complete before this task runs
	Deps []string // Dependencies (task names)
}

// DAGWorkflowInput defines input for DAG workflow execution.
// It contains all tasks to be executed in dependency order.
type DAGWorkflowInput struct {
	// WorkflowID is the unique identifier for this workflow execution
	WorkflowID string

	// Branch is the git branch to execute tasks on
	Branch string

	// Tasks is the list of all tasks to execute in the DAG
	Tasks []Task
}

// ============================================================================
// TCR WORKFLOW TYPES
// ============================================================================

// TCRWorkflowInput defines input for the basic Test-Commit-Revert workflow.
// This workflow follows the TCR pattern: execute a task, run tests, and
// either commit (if tests pass) or revert (if tests fail).
type TCRWorkflowInput struct {
	// CellID is the unique identifier for the isolated execution cell
	CellID string

	// Branch is the git branch to work on
	Branch string

	// TaskID is the unique identifier for this task
	TaskID string

	// Description is a human-readable description of the task
	Description string

	// Prompt is the instruction given to the agent for task execution
	Prompt string
}

// TCRWorkflowResult contains the results of a TCR workflow execution.
// It includes success status, test results, and any errors encountered.
type TCRWorkflowResult struct {
	// Success indicates whether the workflow completed successfully
	Success bool

	// TestsPassed indicates whether the tests passed after task execution
	TestsPassed bool

	// FilesChanged is the list of files modified during task execution
	FilesChanged []string

	// Error contains any error message if the workflow failed
	Error string
}
