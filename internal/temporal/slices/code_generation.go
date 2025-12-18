// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// code_generation.go: Complete vertical slice for code generation operations
// - Test generation (TDD RED phase)
// - Implementation generation (TDD GREEN phase)
// - Code modification and refactoring
//
// This slice follows CUPID principles:
// - Composable: Self-contained code generation operations
// - Unix philosophy: Does code generation, nothing else
// - Predictable: Clear generation tasks with file tracking
// - Idiomatic: TDD workflow, Temporal patterns
// - Domain-centric: Organized around code generation capability
package slices

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
)

// ============================================================================
// ACTIVITIES
// ============================================================================

// CodeGenerationActivities handles all code generation operations
type CodeGenerationActivities struct {
	// No external dependencies - uses SDK client from bootstrap output
}

// NewCodeGenerationActivities creates a new code generation activities instance
func NewCodeGenerationActivities() *CodeGenerationActivities {
	return &CodeGenerationActivities{}
}

// codeGenParams holds parameters for code generation operations.
type codeGenParams struct {
	gateName      string
	agentName     string
	titlePrefix   string
	heartbeatMsg  string
	logStartMsg   string
	logCompleteMsg string
	errorMsg      string
	successMsgFmt string
	promptBuilder func(TaskInput) string
}

// generateCode is a helper that handles common code generation logic.
func (c *CodeGenerationActivities) generateCode(ctx context.Context, output BootstrapOutput, taskInput TaskInput, params codeGenParams) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(params.logStartMsg, "cellID", output.CellID, "taskID", taskInput.TaskID)

	activity.RecordHeartbeat(ctx, params.heartbeatMsg)

	startTime := time.Now()

	client := ReconstructClient(output)
	prompt := params.promptBuilder(taskInput)

	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Agent: "build",
		Title: fmt.Sprintf("%s - %s", params.titlePrefix, taskInput.TaskID),
	})
	if err != nil {
		return &GateResult{
			GateName:      params.gateName,
			Passed:        false,
			Duration:      time.Since(startTime),
			Error:         err.Error(),
			RetryAttempts: 0,
		}, fmt.Errorf("%s in cell %q: %w", params.errorMsg, output.CellID, err)
	}

	duration := time.Since(startTime)
	filesModified := extractModifiedFiles(result)

	logger.Info(params.logCompleteMsg,
		"cellID", output.CellID,
		"filesModified", len(filesModified),
		"duration", duration)

	return &GateResult{
		GateName: params.gateName,
		Passed:   true,
		Duration: duration,
		Message:  fmt.Sprintf(params.successMsgFmt, taskInput.TaskID, len(filesModified)),
		AgentResults: []AgentResult{{
			AgentName:    params.agentName,
			Prompt:       prompt,
			Response:     result.GetText(),
			Success:      true,
			Duration:     duration,
			FilesChanged: filesModified,
		}},
	}, nil
}

// GenerateTests generates test code (TDD RED phase - Gate 1)
//
// This activity implements the first gate of Enhanced TCR:
// 1. Agent generates test code based on acceptance criteria
// 2. Tests must compile
// 3. Tests must fail (because implementation doesn't exist)
//
// Returns GateResult indicating success/failure of test generation.
func (c *CodeGenerationActivities) GenerateTests(ctx context.Context, output BootstrapOutput, taskInput TaskInput) (*GateResult, error) {
	return c.generateCode(ctx, output, taskInput, codeGenParams{
		gateName:       "generate_tests",
		agentName:      "test-generator",
		titlePrefix:    "Generate Tests",
		heartbeatMsg:   "generating test code",
		logStartMsg:    "Generating tests",
		logCompleteMsg: "Test generation completed",
		errorMsg:       "failed to generate tests",
		successMsgFmt:  "Generated tests for %s (%d files modified)",
		promptBuilder:  buildTestGenerationPrompt,
	})
}

// GenerateImplementation generates implementation code (TDD GREEN phase - Gate 4)
//
// This activity implements the fourth gate of Enhanced TCR:
// 1. Agent generates implementation based on failing tests
// 2. Implementation must compile
// 3. Implementation should make tests pass
//
// Returns GateResult indicating success/failure of implementation generation.
func (c *CodeGenerationActivities) GenerateImplementation(ctx context.Context, output BootstrapOutput, taskInput TaskInput) (*GateResult, error) {
	return c.generateCode(ctx, output, taskInput, codeGenParams{
		gateName:       "generate_implementation",
		agentName:      "implementation-generator",
		titlePrefix:    "Generate Implementation",
		heartbeatMsg:   "generating implementation code",
		logStartMsg:    "Generating implementation",
		logCompleteMsg: "Implementation generation completed",
		errorMsg:       "failed to generate implementation",
		successMsgFmt:  "Generated implementation for %s (%d files modified)",
		promptBuilder:  buildImplementationPrompt,
	})
}

// ExecuteGenericTask executes a generic code generation task
//
// This activity provides a flexible code generation interface:
// - Accepts arbitrary prompts
// - Tracks file modifications
// - Returns structured results
//
// Used for tasks that don't fit the standard TDD workflow.
func (c *CodeGenerationActivities) ExecuteGenericTask(ctx context.Context, output BootstrapOutput, taskInput TaskInput) (*TaskOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing generic task", "cellID", output.CellID, "taskID", taskInput.TaskID)

	activity.RecordHeartbeat(ctx, "executing task")

	// Reconstruct SDK client
	client := ReconstructClient(output)

	// Execute task via SDK
	result, err := client.ExecutePrompt(ctx, taskInput.Prompt, &agent.PromptOptions{
		Agent: "build",
		Title: taskInput.TaskID,
	})
	if err != nil {
		return &TaskOutput{
			Success: false,
			Error:   err.Error(),
		}, fmt.Errorf("failed to execute task in cell %q: %w", output.CellID, err)
	}

	// Extract files modified
	filesModified := extractModifiedFiles(result)

	logger.Info("Generic task completed",
		"cellID", output.CellID,
		"taskID", taskInput.TaskID,
		"filesModified", len(filesModified))

	return &TaskOutput{
		Success:       true,
		Output:        result.GetText(),
		FilesModified: filesModified,
	}, nil
}

// GenerateTestsWithRetry generates tests with retry budget
//
// Test generation may fail due to:
// - Ambiguous requirements
// - Complex test scenarios
// - Compilation errors
//
// This activity retries test generation up to maxRetries times.
func (c *CodeGenerationActivities) GenerateTestsWithRetry(ctx context.Context, output BootstrapOutput, taskInput TaskInput, maxRetries int) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating tests with retry", "cellID", output.CellID, "maxRetries", maxRetries)

	var lastResult *GateResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying test generation", "cellID", output.CellID, "attempt", attempt)
			activity.RecordHeartbeat(ctx, fmt.Sprintf("retry attempt %d/%d", attempt, maxRetries))
		}

		result, err := c.GenerateTests(ctx, output, taskInput)
		if err != nil {
			lastErr = err
			lastResult = result
			continue
		}

		if result.Passed {
			result.RetryAttempts = attempt
			return result, nil
		}

		lastResult = result
		lastErr = fmt.Errorf("test generation failed: %s", result.Error)
	}

	// All retries exhausted
	logger.Warn("Test generation failed after retries", "cellID", output.CellID, "attempts", maxRetries+1)

	if lastResult != nil {
		lastResult.RetryAttempts = maxRetries
	}

	return lastResult, lastErr
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// buildTestGenerationPrompt creates a prompt for test generation
func buildTestGenerationPrompt(taskInput TaskInput) string {
	prompt := fmt.Sprintf(`Generate test code for the following task:

Task ID: %s
Description: %s

Requirements:
1. Follow TDD (Test-Driven Development) - write tests FIRST
2. Tests should be comprehensive and cover edge cases
3. Use testify/assert for assertions
4. Tests should fail initially (RED state)
5. Follow Go testing conventions

Prompt: %s

Generate the test code now.`, taskInput.TaskID, taskInput.Description, taskInput.Prompt)

	return prompt
}

// buildImplementationPrompt creates a prompt for implementation generation
func buildImplementationPrompt(taskInput TaskInput) string {
	prompt := fmt.Sprintf(`Generate implementation code to make the failing tests pass:

Task ID: %s
Description: %s

Requirements:
1. Implement minimal code to make tests pass (GREEN state)
2. Follow Go best practices and idioms
3. Write clean, maintainable code
4. Add appropriate error handling
5. Follow existing code patterns in the codebase

Prompt: %s

Generate the implementation code now.`, taskInput.TaskID, taskInput.Description, taskInput.Prompt)

	return prompt
}

// extractModifiedFiles extracts list of modified files from prompt result
//
// This examines tool results to find file write operations.
// Returns list of file paths that were modified.
func extractModifiedFiles(result *agent.PromptResult) []string {
	var files []string

	// Extract from tool results
	toolResults := result.GetToolResults()
	for _, tool := range toolResults {
		if tool.ToolName == "Write" || tool.ToolName == "Edit" {
			// Tool result should contain file path
			// This is a simplified extraction - production code should parse properly
			if tool.ToolResult != nil {
				// Extract file path from tool result
				// For now, just note that files were modified
				files = append(files, "modified")
			}
		}
	}

	// If no tool results, try to extract from response text
	if len(files) == 0 {
		text := result.GetText()
		if containsString(text, "created") || containsString(text, "modified") {
			files = append(files, "modified")
		}
	}

	return files
}
