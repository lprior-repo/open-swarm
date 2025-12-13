// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"open-swarm/internal/agent"
)

// HeadlessExecutor implements the Executor interface using internal/agent.Client.
// It provides a headless execution environment for OpenCode tasks without UI interaction.
type HeadlessExecutor struct {
	client agent.ClientInterface
	config Config
}

// NewHeadlessExecutor creates a new headless executor with the given client and config.
func NewHeadlessExecutor(client agent.ClientInterface, config Config) *HeadlessExecutor {
	// Set defaults if not specified
	if config.MaxTurns <= 0 {
		config.MaxTurns = 10
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.Provider == "" {
		config.Provider = "claude"
	}

	return &HeadlessExecutor{
		client: client,
		config: config,
	}
}

// Execute runs a task through the OpenCode agent and returns the result.
// It implements the Executor interface.
func (e *HeadlessExecutor) Execute(ctx context.Context, req TaskRequest) TaskResult {
	// 1. Validate task request
	if err := e.validateRequest(&req); err != nil {
		return TaskResult{
			Success: false,
			Error:   fmt.Sprintf("validation failed: %v", err),
		}
	}

	// 2. Apply defaults
	maxTurns := req.MaxTurns
	if maxTurns <= 0 {
		maxTurns = e.config.MaxTurns
	}

	// 3. Create timeout context
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	// 4. Create agent session and build prompt with file context
	prompt := e.buildPrompt(&req)

	// 5. Execute prompt with agent
	promptOpts := &agent.PromptOptions{
		Title: "OpenCode Task Execution",
		Agent: "build", // Use build agent for code tasks
	}

	// Set model based on provider
	if req.Provider != "" {
		promptOpts.Model = e.getModelForProvider(req.Provider)
	} else if e.config.Provider != "" {
		promptOpts.Model = e.getModelForProvider(e.config.Provider)
	}

	result, err := e.client.ExecutePrompt(ctx, prompt, promptOpts)
	if err != nil {
		return TaskResult{
			Success: false,
			Error:   fmt.Sprintf("prompt execution failed: %v", err),
		}
	}

	// 6. Monitor turn count (for now we execute once, multi-turn would need session iteration)
	// Note: The current implementation executes a single turn.
	// Multi-turn execution would require iterating with the same sessionID.
	turnCount := 1
	if turnCount > maxTurns {
		return TaskResult{
			Success: false,
			Error:   fmt.Sprintf("exceeded maximum turns (%d)", maxTurns),
		}
	}

	// 7. Extract result text
	output := result.GetText()

	// 8. Get changed files
	filesChanged, err := e.getChangedFiles(ctx)
	if err != nil {
		// Don't fail the task, just log the error in output
		output += fmt.Sprintf("\n\nWarning: Could not retrieve file status: %v", err)
	}

	// 9. Return structured TaskResult
	return TaskResult{
		Success:      true,
		Output:       output,
		FilesChanged: filesChanged,
		Error:        "",
	}
}

// validateRequest validates the task request parameters
func (e *HeadlessExecutor) validateRequest(req *TaskRequest) error {
	if req.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(req.Prompt) > 50000 {
		return fmt.Errorf("prompt exceeds maximum length of 50000 characters")
	}
	if req.MaxTurns < 0 {
		return fmt.Errorf("maxTurns cannot be negative")
	}
	return nil
}

// buildPrompt constructs the full prompt with file context
func (e *HeadlessExecutor) buildPrompt(req *TaskRequest) string {
	var sb strings.Builder

	// Add file context if provided
	if len(req.Files) > 0 {
		sb.WriteString("Files for context:\n")
		for _, file := range req.Files {
			sb.WriteString(fmt.Sprintf("- %s\n", file))
		}
		sb.WriteString("\n")
	}

	// Add main prompt
	sb.WriteString(req.Prompt)

	return sb.String()
}

// getModelForProvider maps provider names to OpenCode model IDs
func (e *HeadlessExecutor) getModelForProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "claude":
		return "anthropic/claude-sonnet-4-5"
	case "copilot":
		return "github/copilot"
	default:
		return "anthropic/claude-sonnet-4-5" // Default to Claude
	}
}

// getChangedFiles retrieves the list of files that were modified during execution
func (e *HeadlessExecutor) getChangedFiles(ctx context.Context) ([]string, error) {
	fileStatus, err := e.client.GetFileStatus(ctx)
	if err != nil {
		return nil, err
	}

	filesChanged := make([]string, 0, len(fileStatus))
	for _, file := range fileStatus {
		if file.Path != "" {
			filesChanged = append(filesChanged, file.Path)
		}
	}

	return filesChanged, nil
}
