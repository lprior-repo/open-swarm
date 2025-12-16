// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
	"time"

	"open-swarm/internal/agent"
)

// ExecutorConfig configures the OpenCode executor behavior
type ExecutorConfig struct {
	MaxTurns int           // Maximum number of turns in a conversation
	Timeout  time.Duration // Timeout for a single prompt execution
}

// ExecutorImpl wraps the OpenCode agent client with execution logic
type ExecutorImpl struct {
	client agent.ClientInterface
	config ExecutorConfig
}

// NewExecutor creates a new OpenCode executor
func NewExecutor(client agent.ClientInterface, config ExecutorConfig) *ExecutorImpl {
	if config.MaxTurns <= 0 {
		config.MaxTurns = 10 // Default max turns
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Minute // Default timeout
	}
	return &ExecutorImpl{
		client: client,
		config: config,
	}
}

// ExecuteRequest represents a task execution request
type ExecuteRequest struct {
	TaskID      string
	Description string
	Prompt      string
	SessionID   string // Optional: reuse existing session
}

// ExecuteResponse represents the result of task execution
type ExecuteResponse struct {
	Success       bool
	Output        string
	FilesModified []string
	Turns         int
	SessionID     string
	ErrorMessage  string
}

// Validate checks if the request is valid
func (r *ExecuteRequest) Validate() error {
	if r.TaskID == "" {
		return fmt.Errorf("TaskID is required")
	}
	if r.Prompt == "" {
		return fmt.Errorf("Prompt is required")
	}
	if len(r.Prompt) > 10000 {
		return fmt.Errorf("prompt exceeds maximum length of 10000 characters")
	}
	return nil
}

// Execute runs a task through the OpenCode agent
func (e *ExecutorImpl) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return &ExecuteResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("invalid request: %v", err),
		}, nil // Return as validation error, not execution error
	}

	// Create a timeout context if one isn't already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	// Track execution state
	turn := 0
	sessionID := req.SessionID
	var allOutput string
	var lastSessionID string

	// Execute the prompt
	promptOpts := &agent.PromptOptions{
		Title:     fmt.Sprintf("Task: %s", req.TaskID),
		SessionID: sessionID,
		Agent:     "build",
	}

	turn++

	// Check turn limit before execution
	if turn > e.config.MaxTurns {
		return &ExecuteResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("exceeded maximum turns (%d)", e.config.MaxTurns),
			Turns:        turn,
			SessionID:    lastSessionID,
		}, nil
	}

	result, err := e.client.ExecutePrompt(ctx, req.Prompt, promptOpts)
	if err != nil {
		return &ExecuteResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("prompt execution failed: %v", err),
			Turns:        turn,
			SessionID:    lastSessionID,
		}, nil
	}

	lastSessionID = result.SessionID
	allOutput = result.GetText()

	// Get file modifications
	fileStatus, err := e.client.GetFileStatus(ctx)
	if err != nil {
		return &ExecuteResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get file status: %v", err),
			Turns:        turn,
			SessionID:    lastSessionID,
		}, nil
	}

	filesModified := make([]string, 0)
	for _, file := range fileStatus {
		if file.Path != "" {
			filesModified = append(filesModified, file.Path)
		}
	}

	return &ExecuteResponse{
		Success:       true,
		Output:        allOutput,
		FilesModified: filesModified,
		Turns:         turn,
		SessionID:     lastSessionID,
	}, nil
}
