// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"open-swarm/internal/telemetry"
)

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

// Client wraps the OpenCode SDK client with reactor-specific functionality
// Enforces INV-004: SDK Client must be configured with specific BaseURL (localhost:PORT)
// Enforces INV-006: Command execution must use SDK 'client.Command.Execute'
type Client struct {
	sdk     *opencode.Client
	baseURL string
	port    int
}

// NewClient creates a new OpenCode SDK client configured for a specific server instance
// INV-004: SDK Client must be configured with specific BaseURL (localhost:PORT)
func NewClient(baseURL string, port int) *Client {
	// Configure SDK to connect to local opencode serve instance
	sdk := opencode.NewClient(
		option.WithBaseURL(baseURL),
		// No API key needed for local connections
	)

	return &Client{
		sdk:     sdk,
		baseURL: baseURL,
		port:    port,
	}
}

// GetSDK returns the underlying OpenCode SDK client
func (c *Client) GetSDK() *opencode.Client {
	return c.sdk
}

// GetBaseURL returns the base URL this client is connected to
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetPort returns the port this client is connected to
func (c *Client) GetPort() int {
	return c.port
}

// ExecutePrompt sends a prompt to the OpenCode server and returns the response
// This is a high-level wrapper around the SDK's session/message APIs with OpenTelemetry tracing
func (c *Client) ExecutePrompt(ctx context.Context, prompt string, opts *PromptOptions) (*PromptResult, error) {
	// Start tracing span
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "ExecutePrompt",
		trace.WithAttributes(
			attribute.String("opencode.base_url", c.baseURL),
			attribute.Int("opencode.port", c.port),
			attribute.Int("prompt.length", len(prompt)),
		),
	)
	defer span.End()

	startTime := time.Now()
	if opts == nil {
		opts = &PromptOptions{}
	}

	// Add prompt details to trace
	if opts.Model != "" {
		span.SetAttributes(attribute.String("opencode.model", opts.Model))
	}
	if opts.Agent != "" {
		span.SetAttributes(attribute.String("opencode.agent", opts.Agent))
	}
	if opts.Title != "" {
		span.SetAttributes(attribute.String("opencode.title", opts.Title))
	}

	// Record prompt start event
	telemetry.AddEvent(ctx, "prompt.start", attribute.String("prompt_preview", truncateString(prompt, 100)))

	sessionID, err := c.getOrCreateSession(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create session")
		telemetry.AddEvent(ctx, "session.create.failed", telemetry.ErrorAttrs(err)...)
		return nil, err
	}

	span.SetAttributes(attribute.String("opencode.session_id", sessionID))
	telemetry.AddEvent(ctx, "session.created", attribute.String("session_id", sessionID))

	message, err := c.sendPromptMessage(ctx, sessionID, prompt, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send prompt")
		telemetry.AddEvent(ctx, "prompt.send.failed", telemetry.ErrorAttrs(err)...)
		return nil, err
	}

	result := c.extractPromptResult(sessionID, message)
	duration := time.Since(startTime)

	// Add result metrics to trace
	span.SetAttributes(
		attribute.Int("opencode.response_parts", len(result.Parts)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Bool("success", true),
	)

	// Record completion event with metrics
	telemetry.AddEvent(ctx, "prompt.completed",
		attribute.String("session_id", sessionID),
		attribute.String("message_id", result.MessageID),
		attribute.Int("response_parts", len(result.Parts)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	span.SetStatus(codes.Ok, "prompt executed successfully")
	return result, nil
}

func (c *Client) getOrCreateSession(ctx context.Context, opts *PromptOptions) (string, error) {
	if opts.SessionID != "" {
		telemetry.AddEvent(ctx, "session.reused", attribute.String("session_id", opts.SessionID))
		return opts.SessionID, nil
	}

	telemetry.AddEvent(ctx, "session.creating", attribute.String("title", opts.Title))
	session, err := c.sdk.Session.New(ctx, opencode.SessionNewParams{
		Title: opencode.F(opts.Title),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return session.ID, nil
}

func (c *Client) sendPromptMessage(ctx context.Context, sessionID string, prompt string, opts *PromptOptions) (*opencode.SessionPromptResponse, error) {
	telemetry.AddEvent(ctx, "prompt.sending",
		attribute.String("session_id", sessionID),
		attribute.Int("prompt_length", len(prompt)),
	)

	parts := []opencode.SessionPromptParamsPartUnion{
		opencode.TextPartInputParam{
			Type: opencode.F(opencode.TextPartInputTypeText),
			Text: opencode.F(prompt),
		},
	}

	promptParams := opencode.SessionPromptParams{
		Parts: opencode.F(parts),
	}

	c.applyPromptOptions(&promptParams, opts)

	message, err := c.sdk.Session.Prompt(ctx, sessionID, promptParams)
	if err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	telemetry.AddEvent(ctx, "prompt.sent",
		attribute.String("session_id", sessionID),
		attribute.String("message_id", message.Info.ID),
	)

	return message, nil
}

func (c *Client) applyPromptOptions(promptParams *opencode.SessionPromptParams, opts *PromptOptions) {
	if opts.Model != "" {
		// Parse model string in format "provider/model" or just "model"
		providerID := ""
		modelID := opts.Model

		if strings.Contains(opts.Model, "/") {
			parts := strings.SplitN(opts.Model, "/", 2)
			if len(parts) == 2 {
				providerID = parts[0]
				modelID = parts[1]
			}
		}

		promptParams.Model = opencode.F(opencode.SessionPromptParamsModel{
			ProviderID: opencode.F(providerID),
			ModelID:    opencode.F(modelID),
		})
	}

	if opts.Agent != "" {
		promptParams.Agent = opencode.F(opts.Agent)
	}

	if opts.NoReply {
		promptParams.NoReply = opencode.F(true)
	}
}

func (c *Client) extractPromptResult(sessionID string, message *opencode.SessionPromptResponse) *PromptResult {
	result := &PromptResult{
		SessionID: sessionID,
		MessageID: message.Info.ID,
		Parts:     make([]ResultPart, 0, len(message.Parts)),
	}

	textParts := 0
	toolParts := 0
	reasoningParts := 0

	for _, part := range message.Parts {
		resultPart := ResultPart{
			Type: string(part.Type),
		}

		switch part.Type {
		case opencode.PartTypeText:
			resultPart.Text = part.Text
			textParts++
		case opencode.PartTypeTool:
			resultPart.ToolName = part.Tool
			toolParts++
		case opencode.PartTypeReasoning:
			resultPart.Text = part.Text
			reasoningParts++
		}

		result.Parts = append(result.Parts, resultPart)
	}

	return result
}

// ExecuteCommand executes a command (slash command) on the OpenCode server with OpenTelemetry tracing
// INV-006: Command execution must use SDK
func (c *Client) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*PromptResult, error) {
	// Start tracing span
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "ExecuteCommand",
		trace.WithAttributes(
			attribute.String("opencode.base_url", c.baseURL),
			attribute.Int("opencode.port", c.port),
			attribute.String("opencode.command", command),
			attribute.Int("opencode.args_count", len(args)),
			attribute.String("opencode.session_id", sessionID),
		),
	)
	defer span.End()

	startTime := time.Now()

	// Record command start event
	telemetry.AddEvent(ctx, "command.start",
		attribute.String("command", command),
		attribute.StringSlice("args", args),
	)

	// Convert args array to space-separated string
	argsStr := ""
	if len(args) > 0 {
		argsStr = strings.Join(args, " ")
		span.SetAttributes(attribute.String("opencode.args", argsStr))
	}

	cmdParams := opencode.SessionCommandParams{
		Command:   opencode.F(command),
		Arguments: opencode.F(argsStr),
	}

	response, err := c.sdk.Session.Command(ctx, sessionID, cmdParams)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute command")
		telemetry.AddEvent(ctx, "command.failed", telemetry.ErrorAttrs(err)...)
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	duration := time.Since(startTime)

	// Extract results from command response
	result := &PromptResult{
		SessionID: sessionID,
		MessageID: response.Info.ID,
		Parts:     make([]ResultPart, 0, len(response.Parts)),
	}

	textParts := 0
	toolParts := 0
	fileParts := 0

	for _, part := range response.Parts {
		resultPart := ResultPart{
			Type: string(part.Type),
		}

		switch part.Type {
		case opencode.PartTypeText:
			resultPart.Text = part.Text
			textParts++
		case opencode.PartTypeTool:
			resultPart.ToolName = part.Tool
			toolParts++
			// Tool result would need additional handling if available
		case opencode.PartTypeReasoning:
			// Reasoning part - store as text for now
			resultPart.Text = part.Text
		case opencode.PartTypeFile:
			fileParts++
			// File part handling
		case opencode.PartTypeStepStart:
			// Step start marker
		case opencode.PartTypeStepFinish:
			// Step finish marker
		case opencode.PartTypeSnapshot:
			// Snapshot part
		case opencode.PartTypePatch:
			// Patch part
		case opencode.PartTypeAgent:
			// Agent part
		case opencode.PartTypeRetry:
			// Retry part
		}

		result.Parts = append(result.Parts, resultPart)
	}

	// Add metrics to span
	span.SetAttributes(
		attribute.Int("opencode.response_parts", len(result.Parts)),
		attribute.Int("opencode.text_parts", textParts),
		attribute.Int("opencode.tool_parts", toolParts),
		attribute.Int("opencode.file_parts", fileParts),
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Bool("success", true),
	)

	// Record completion event
	telemetry.AddEvent(ctx, "command.completed",
		attribute.String("command", command),
		attribute.String("message_id", response.Info.ID),
		attribute.Int("response_parts", len(result.Parts)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	span.SetStatus(codes.Ok, "command executed successfully")
	return result, nil
}

// ListSessions returns all sessions on the server
func (c *Client) ListSessions(ctx context.Context) ([]opencode.Session, error) {
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "ListSessions")
	defer span.End()

	sessions, err := c.sdk.Session.List(ctx, opencode.SessionListParams{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list sessions")
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	if sessions == nil {
		span.SetStatus(codes.Ok, "no sessions found")
		return []opencode.Session{}, nil
	}
	span.SetAttributes(attribute.Int("sessions.count", len(*sessions)))
	span.SetStatus(codes.Ok, "sessions listed successfully")
	return *sessions, nil
}

// GetSession retrieves a specific session
func (c *Client) GetSession(ctx context.Context, sessionID string) (*opencode.Session, error) {
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "GetSession",
		trace.WithAttributes(attribute.String("opencode.session_id", sessionID)),
	)
	defer span.End()

	session, err := c.sdk.Session.Get(ctx, sessionID, opencode.SessionGetParams{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get session")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	span.SetStatus(codes.Ok, "session retrieved successfully")
	return session, nil
}

// DeleteSession deletes a session
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "DeleteSession",
		trace.WithAttributes(attribute.String("opencode.session_id", sessionID)),
	)
	defer span.End()

	_, err := c.sdk.Session.Delete(ctx, sessionID, opencode.SessionDeleteParams{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}
	span.SetStatus(codes.Ok, "session deleted successfully")
	telemetry.AddEvent(ctx, "session.deleted", attribute.String("session_id", sessionID))
	return nil
}

// truncateString truncates a string to the specified length, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// AbortSession aborts a running session
func (c *Client) AbortSession(ctx context.Context, sessionID string) error {
	ctx, span := telemetry.StartSpan(ctx, "opencode.client", "AbortSession",
		trace.WithAttributes(attribute.String("opencode.session_id", sessionID)),
	)
	defer span.End()

	_, err := c.sdk.Session.Abort(ctx, sessionID, opencode.SessionAbortParams{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to abort session")
		return fmt.Errorf("failed to abort session: %w", err)
	}
	span.SetStatus(codes.Ok, "session aborted successfully")
	telemetry.AddEvent(ctx, "session.aborted", attribute.String("session_id", sessionID))
	return nil
}

// GetFileStatus retrieves the status of tracked files
func (c *Client) GetFileStatus(ctx context.Context) ([]opencode.File, error) {
	files, err := c.sdk.File.Status(ctx, opencode.FileStatusParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file status: %w", err)
	}
	if files == nil {
		return []opencode.File{}, nil
	}
	return *files, nil
}

// ReadFile reads the content of a file
func (c *Client) ReadFile(ctx context.Context, path string) (string, error) {
	file, err := c.sdk.File.Read(ctx, opencode.FileReadParams{
		Path: opencode.F(path),
	})
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return file.Content, nil
}
