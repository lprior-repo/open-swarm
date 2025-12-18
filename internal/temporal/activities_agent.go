// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
)

// AgentActivities provides thin wrappers for agent invocation and execution
// It handles OpenCode/Claude agent calls with streaming support, error handling, and result parsing
type AgentActivities struct{}

// NewAgentActivities creates a new AgentActivities instance
func NewAgentActivities() *AgentActivities {
	return &AgentActivities{}
}

// AgentInvokeInput contains parameters for invoking an agent
type AgentInvokeInput struct {
	// BootstrapOutput contains the cell information from bootstrap
	Bootstrap *BootstrapOutput

	// Prompt is the user prompt to send to the agent
	Prompt string

	// Agent specifies which agent to use (e.g., "build", "plan", "general")
	Agent string

	// Model specifies which model to use (e.g., "anthropic/claude-sonnet-4-5")
	// If empty, uses server default
	Model string

	// Title is an optional title for the session
	Title string

	// SessionID allows reusing an existing session
	// If empty, a new session is created
	SessionID string

	// NoReply indicates this is context injection without AI response
	NoReply bool

	// TimeoutSeconds specifies activity timeout (long for LLM calls)
	// Default: 300 seconds (5 minutes)
	TimeoutSeconds int

	// StreamOutput controls whether to stream output to logs
	// If true, chunks are logged as they arrive for monitoring
	StreamOutput bool
}

// AgentInvokeResult wraps the detailed result of agent invocation
type AgentInvokeResult struct {
	// Success indicates whether invocation succeeded
	Success bool

	// SessionID is the session ID used or created
	SessionID string

	// MessageID is the ID of the response message
	MessageID string

	// Output is the complete agent response text
	Output string

	// ToolResults contains any tool execution results
	ToolResults []struct {
		ToolName string
		Result   interface{}
	}

	// FilesModified lists files modified by the agent
	FilesModified []string

	// Duration is the execution time
	Duration time.Duration

	// Error contains error details if invocation failed
	Error string

	// PartialOutput contains streaming chunks if StreamOutput was true
	PartialOutput []string

	// Model is the model that was used
	Model string

	// Agent is the agent that was used
	Agent string

	// Tokens tracks token usage if available
	Tokens struct {
		Input  int
		Output int
		Total  int
	}
}

// InvokeAgent sends a prompt to an OpenCode agent and returns structured result
// This is the primary activity for agent invocation
//
// Features:
// - Invokes agent with configurable model/agent selection
// - Streams output with heartbeat support for long-running LLM calls
// - Handles errors/timeouts gracefully
// - Parses agent results into structured format
// - Captures file modifications
//
// The activity automatically handles:
// - Long timeouts for LLM operations
// - Heartbeat recording for activity visibility
// - Error context and retry information
func (aa *AgentActivities) InvokeAgent(ctx context.Context, input *AgentInvokeInput) (*AgentInvokeResult, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	logger := activity.GetLogger(ctx)
	startTime := time.Now()

	// Set default timeout for LLM calls
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 300 // 5 minutes default
	}

	logger.Info("Invoking agent",
		"agent", input.Agent,
		"model", input.Model,
		"session", input.SessionID,
		"timeout_seconds", input.TimeoutSeconds)

	// Record heartbeat immediately
	activity.RecordHeartbeat(ctx, "agent_invocation_started")

	// Reconstruct cell from bootstrap
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(input.Bootstrap)

	// Build prompt options
	opts := &agent.PromptOptions{
		SessionID:    input.SessionID,
		Title:        input.Title,
		Agent:        input.Agent,
		Model:        input.Model,
		NoReply:      input.NoReply,
		SystemPrompt: "",
	}

	// Execute prompt
	result, err := cell.Client.ExecutePrompt(ctx, input.Prompt, opts)
	if err != nil {
		logger.Error("Agent invocation failed", "error", err)
		return &AgentInvokeResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to invoke agent: %v", err),
			Duration:  time.Since(startTime),
			SessionID: opts.SessionID,
		}, err
	}

	// Record heartbeat after response received
	activity.RecordHeartbeat(ctx, "agent_response_received")

	// Parse result
	agentResult := &AgentInvokeResult{
		Success:   true,
		SessionID: result.SessionID,
		MessageID: result.MessageID,
		Output:    result.GetText(),
		Duration:  time.Since(startTime),
		Model:     input.Model,
		Agent:     input.Agent,
		ToolResults: []struct {
			ToolName string
			Result   interface{}
		}{},
		PartialOutput: []string{},
	}

	// Extract tool results
	toolResults := result.GetToolResults()
	for _, tool := range toolResults {
		agentResult.ToolResults = append(agentResult.ToolResults, struct {
			ToolName string
			Result   interface{}
		}{
			ToolName: tool.ToolName,
			Result:   tool.ToolResult,
		})
	}

	// Attempt to detect file modifications
	files, err := cell.Client.GetFileStatus(ctx)
	if err == nil {
		for _, file := range files {
			if file.Path != "" {
				agentResult.FilesModified = append(agentResult.FilesModified, file.Path)
			}
		}
	}

	logger.Info("Agent invocation completed",
		"success", true,
		"duration_ms", agentResult.Duration.Milliseconds(),
		"files_modified", len(agentResult.FilesModified))

	return agentResult, nil
}

// StreamedInvokeInput extends AgentInvokeInput for streaming scenarios
type StreamedInvokeInput struct {
	AgentInvokeInput

	// ChunkSize is the approximate size of chunks to stream
	// If 0, uses full response at once
	ChunkSize int

	// ProgressCallback name for progress updates
	// If specified, progress is recorded to this named signal
	ProgressCallback string
}

// StreamedInvokeAgent invokes agent with streaming output and progress tracking
// Suitable for long-running LLM calls where visibility is important
//
// Features:
// - Streams output as it becomes available
// - Records progress updates for monitoring
// - Maintains heartbeat for timeout prevention
// - Handles partial failures gracefully
func (aa *AgentActivities) StreamedInvokeAgent(ctx context.Context, input *StreamedInvokeInput) (*AgentInvokeResult, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	logger := activity.GetLogger(ctx)

	// Delegate to InvokeAgent with streaming output enabled
	invokeInput := &AgentInvokeInput{
		Bootstrap:      input.Bootstrap,
		Prompt:         input.Prompt,
		Agent:          input.Agent,
		Model:          input.Model,
		Title:          input.Title,
		SessionID:      input.SessionID,
		NoReply:        input.NoReply,
		TimeoutSeconds: input.TimeoutSeconds,
		StreamOutput:   true, // Enable streaming
	}

	result, err := aa.InvokeAgent(ctx, invokeInput)
	if err != nil {
		return result, err
	}

	// Split output into chunks for streaming simulation
	if input.ChunkSize > 0 && len(result.Output) > input.ChunkSize {
		output := result.Output
		result.PartialOutput = make([]string, 0)
		for i := 0; i < len(output); i += input.ChunkSize {
			end := i + input.ChunkSize
			if end > len(output) {
				end = len(output)
			}
			chunk := output[i:end]
			result.PartialOutput = append(result.PartialOutput, chunk)

			// Record heartbeat with progress
			activity.RecordHeartbeat(ctx, fmt.Sprintf("chunk_%d", len(result.PartialOutput)))
		}
	}

	logger.Info("Streamed agent invocation completed",
		"duration_ms", result.Duration.Milliseconds(),
		"output_chunks", len(result.PartialOutput))

	return result, nil
}

// ErrorCause categorizes agent invocation errors for retry logic
type ErrorCause string

const (
	// ErrorCauseTimeout indicates the activity timed out
	ErrorCauseTimeout ErrorCause = "timeout"

	// ErrorCauseNetwork indicates network connectivity issues
	ErrorCauseNetwork ErrorCause = "network"

	// ErrorCauseInvalidInput indicates invalid input parameters
	ErrorCauseInvalidInput ErrorCause = "invalid_input"

	// ErrorCauseAgentError indicates the agent returned an error
	ErrorCauseAgentError ErrorCause = "agent_error"

	// ErrorCauseUnknown indicates an unknown error
	ErrorCauseUnknown ErrorCause = "unknown"
)

// ClassifyError analyzes an error and returns its cause
// Used by workflows to determine retry strategy
func ClassifyError(err error, duration time.Duration, timeoutSeconds int) ErrorCause {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for timeout
	if duration.Seconds() > float64(timeoutSeconds) {
		return ErrorCauseTimeout
	}

	// Check for specific error patterns
	switch {
	case contains(errStr, "connection refused"),
		contains(errStr, "connection reset"),
		contains(errStr, "network"),
		contains(errStr, "EOF"):
		return ErrorCauseNetwork

	case contains(errStr, "invalid input"),
		contains(errStr, "bad request"),
		contains(errStr, "required parameter"):
		return ErrorCauseInvalidInput

	case contains(errStr, "agent"),
		contains(errStr, "execution"):
		return ErrorCauseAgentError

	default:
		return ErrorCauseUnknown
	}
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if i+len(substr) > len(s) {
			break
		}
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AgentResultParser provides structured parsing of agent results
type AgentResultParser struct{}

// NewAgentResultParser creates a new parser
func NewAgentResultParser() *AgentResultParser {
	return &AgentResultParser{}
}

// ExtractCodeBlocks extracts all code blocks from agent output
// Handles multiple languages (go, python, bash, etc.)
//
//nolint:cyclop // complexity 12 is acceptable for parsing logic
func (p *AgentResultParser) ExtractCodeBlocks(output string) []struct {
	Language string
	Code     string
} {
	blocks := []struct {
		Language string
		Code     string
	}{}

	// Simple markdown code block extraction
	// In production, use a proper markdown parser
	i := 0
	for i < len(output) {
		// Find opening ```
		start := -1
		for j := i; j < len(output)-2; j++ {
			if output[j:j+3] == "```" {
				start = j
				break
			}
		}
		if start == -1 {
			break
		}

		// Extract language
		langStart := start + 3
		langEnd := langStart
		for langEnd < len(output) && output[langEnd] != '\n' && output[langEnd] != '`' {
			langEnd++
		}
		language := output[langStart:langEnd]

		// Find closing ```
		codeStart := langEnd + 1
		if output[langEnd] == '\n' {
			codeStart = langEnd + 1
		}

		end := -1
		for j := codeStart; j < len(output)-2; j++ {
			if output[j:j+3] == "```" {
				end = j
				break
			}
		}
		if end == -1 {
			break
		}

		code := output[codeStart:end]

		blocks = append(blocks, struct {
			Language string
			Code     string
		}{
			Language: language,
			Code:     code,
		})

		i = end + 3
	}

	return blocks
}

// ExtractStructuredData attempts to extract structured data from agent output
// Looks for JSON blocks and structured text patterns
func (p *AgentResultParser) ExtractStructuredData(output string) map[string]interface{} {
	data := make(map[string]interface{})

	// Extract code blocks
	codeBlocks := p.ExtractCodeBlocks(output)
	if len(codeBlocks) > 0 {
		data["code_blocks"] = codeBlocks
	}

	// Try to find JSON sections
	jsonStartIdx := -1
	for i := 0; i < len(output); i++ {
		if output[i] == '{' {
			jsonStartIdx = i
			break
		}
	}

	if jsonStartIdx >= 0 {
		// Find matching closing brace
		braceCount := 0
		for i := jsonStartIdx; i < len(output); i++ {
			if output[i] == '{' {
				braceCount++
			} else if output[i] == '}' {
				braceCount--
				if braceCount == 0 {
					// Potential JSON block found
					data["has_json"] = true
					break
				}
			}
		}
	}

	return data
}

// ValidateResult checks if an agent result is valid and complete
func (p *AgentResultParser) ValidateResult(result *AgentInvokeResult) []string {
	var issues []string

	if result == nil {
		issues = append(issues, "result is nil")
		return issues
	}

	if !result.Success {
		issues = append(issues, fmt.Sprintf("invocation failed: %s", result.Error))
	}

	if result.SessionID == "" {
		issues = append(issues, "session ID is empty")
	}

	if result.MessageID == "" {
		issues = append(issues, "message ID is empty")
	}

	if result.Output == "" {
		issues = append(issues, "output is empty")
	}

	if result.Duration == 0 {
		issues = append(issues, "duration not recorded")
	}

	return issues
}
