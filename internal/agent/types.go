package agent

// PromptOptions configures how a prompt is executed
type PromptOptions struct {
	// SessionID to use (if empty, creates new session)
	SessionID string

	// Title for new session (if SessionID is empty)
	Title string

	// Model to use for this prompt (e.g., "anthropic/claude-sonnet-4-5")
	Model string

	// Agent to use (e.g., "build", "plan", "general")
	Agent string

	// NoReply indicates this is context injection without AI response
	NoReply bool

	// SystemPrompt overrides the system prompt
	SystemPrompt string

	// Tools to enable for this prompt
	Tools []string
}

// PromptResult contains the result of a prompt execution
type PromptResult struct {
	SessionID string
	MessageID string
	Parts     []ResultPart
}

// ResultPart represents a part of the response
type ResultPart struct {
	Type       string // "text", "tool", etc.
	Text       string
	ToolName   string
	ToolResult interface{}
}

// GetText returns all text parts concatenated
func (r *PromptResult) GetText() string {
	var text string
	for _, part := range r.Parts {
		if part.Type == "text" {
			text += part.Text
		}
	}
	return text
}

// GetToolResults returns all tool execution results
func (r *PromptResult) GetToolResults() []ResultPart {
	var tools []ResultPart
	for _, part := range r.Parts {
		if part.Type == "tool" {
			tools = append(tools, part)
		}
	}
	return tools
}

// TaskContext represents the context for a task execution
type TaskContext struct {
	TaskID      string
	Description string
	Files       []string
	Prompt      string
}

// ExecutionResult represents the result of a task execution
type ExecutionResult struct {
	Success        bool
	Output         string
	FilesModified  []string
	TestsPassed    bool
	ErrorMessage   string
	SessionID      string
}
