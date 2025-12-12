package agent

import (
	"context"
	"fmt"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

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
// This is a high-level wrapper around the SDK's session/message APIs
func (c *Client) ExecutePrompt(ctx context.Context, prompt string, opts *PromptOptions) (*PromptResult, error) {
	if opts == nil {
		opts = &PromptOptions{}
	}

	// 1. Create or get session
	var sessionID string
	if opts.SessionID != "" {
		sessionID = opts.SessionID
	} else {
		session, err := c.sdk.Session.New(ctx, opencode.SessionNewParams{
			Title: opencode.F(opts.Title),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = session.ID
	}

	// 2. Send prompt as message
	parts := []opencode.SessionPromptParamsPartUnion{
		opencode.TextPartInputParam{
			Type: opencode.F(opencode.TextPartInputTypeText),
			Text: opencode.F(prompt),
		},
	}

	promptParams := opencode.SessionPromptParams{
		Parts: opencode.F(parts),
	}

	// Set model if specified
	if opts.Model != "" {
		promptParams.Model = opencode.F(opencode.SessionPromptParamsModel{
			ModelID: opencode.F(opts.Model),
		})
	}

	// Set agent if specified
	if opts.Agent != "" {
		promptParams.Agent = opencode.F(opts.Agent)
	}

	// Set noReply if specified (for context injection without AI response)
	if opts.NoReply {
		promptParams.NoReply = opencode.F(true)
	}

	message, err := c.sdk.Session.Prompt(ctx, sessionID, promptParams)
	if err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	// 3. Extract response
	result := &PromptResult{
		SessionID: sessionID,
		MessageID: message.Info.ID,
		Parts:     make([]ResultPart, 0, len(message.Parts)),
	}

	for _, part := range message.Parts {
		resultPart := ResultPart{
			Type: string(part.Type),
		}

		switch part.Type {
		case opencode.PartTypeText:
			resultPart.Text = part.Text
		case opencode.PartTypeTool:
			resultPart.ToolName = part.Tool
			// Tool result would need additional handling if available
		}

		result.Parts = append(result.Parts, resultPart)
	}

	return result, nil
}

// ExecuteCommand executes a command (slash command) on the OpenCode server
// INV-006: Command execution must use SDK
func (c *Client) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*PromptResult, error) {
	// Convert args array to space-separated string
	argsStr := ""
	if len(args) > 0 {
		for i, arg := range args {
			if i > 0 {
				argsStr += " "
			}
			argsStr += arg
		}
	}

	cmdParams := opencode.SessionCommandParams{
		Command:   opencode.F(command),
		Arguments: opencode.F(argsStr),
	}

	message, err := c.sdk.Session.Command(ctx, sessionID, cmdParams)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	result := &PromptResult{
		SessionID: sessionID,
		MessageID: message.Info.ID,
		Parts:     make([]ResultPart, 0, len(message.Parts)),
	}

	for _, part := range message.Parts {
		resultPart := ResultPart{
			Type: string(part.Type),
		}

		switch part.Type {
		case opencode.PartTypeText:
			resultPart.Text = part.Text
		case opencode.PartTypeTool:
			resultPart.ToolName = part.Tool
			// Tool result would need additional handling if available
		}

		result.Parts = append(result.Parts, resultPart)
	}

	return result, nil
}

// ListSessions returns all sessions on the server
func (c *Client) ListSessions(ctx context.Context) ([]opencode.Session, error) {
	sessions, err := c.sdk.Session.List(ctx, opencode.SessionListParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	if sessions == nil {
		return []opencode.Session{}, nil
	}
	return *sessions, nil
}

// GetSession retrieves a specific session
func (c *Client) GetSession(ctx context.Context, sessionID string) (*opencode.Session, error) {
	session, err := c.sdk.Session.Get(ctx, sessionID, opencode.SessionGetParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// DeleteSession deletes a session
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := c.sdk.Session.Delete(ctx, sessionID, opencode.SessionDeleteParams{})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// AbortSession aborts a running session
func (c *Client) AbortSession(ctx context.Context, sessionID string) error {
	_, err := c.sdk.Session.Abort(ctx, sessionID, opencode.SessionAbortParams{})
	if err != nil {
		return fmt.Errorf("failed to abort session: %w", err)
	}
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
