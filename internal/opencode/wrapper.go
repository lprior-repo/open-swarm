// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"time"
)

// ExecutorInterface defines the interface for executing standardized tasks via OpenCode.
// It provides a thin wrapper over the underlying OpenCode client to normalize
// task execution across different agent types and providers.
type ExecutorInterface interface {
	// Execute runs a task with the given request and returns the result.
	// The context is used to manage cancellation and timeouts.
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
}

// TaskRequest encapsulates all parameters needed to execute a task.
// It standardizes task input across different execution contexts and providers.
type TaskRequest struct {
	// Prompt is the main instruction or question for the AI agent.
	Prompt string

	// Files is a list of file paths that should be included as context.
	// Empty slice if no files are needed.
	Files []string

	// MaxTurns specifies the maximum number of agent turns allowed.
	// A turn represents one round of AI reasoning and execution.
	MaxTurns int

	// Provider specifies which AI provider to use (e.g., "claude", "copilot").
	Provider string
}

// TaskResult contains the outcome of a task execution.
// It aggregates success indicators, output, and any changes made to the filesystem.
type TaskResult struct {
	// Success indicates whether the task completed successfully.
	Success bool

	// Output is the text output or response from the execution.
	Output string

	// FilesChanged lists the paths of files that were modified during execution.
	FilesChanged []string

	// Error contains the error message if the task failed.
	// Empty string if no error occurred.
	Error string
}

// Config holds configuration for task execution behavior.
// It standardizes settings like provider selection, feature flags, and timeouts.
type Config struct {
	// Provider specifies the default AI provider to use ("claude" or "copilot").
	Provider string

	// SerenaEnabled controls whether Serena memory integration is enabled.
	SerenaEnabled bool

	// MaxTurns sets the default maximum number of agent turns per task.
	MaxTurns int

	// Timeout is the maximum duration allowed for task execution.
	// Execution will be cancelled if this duration is exceeded.
	Timeout time.Duration
}
