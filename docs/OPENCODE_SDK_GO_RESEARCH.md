# OpenCode SDK for Go - Comprehensive Research & Ecosystem Guide

**Last Updated**: December 2025  
**SDK Version**: v0.19.2  
**Go Requirements**: 1.22+

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [OpenCode Ecosystem Overview](#opencode-ecosystem-overview)
3. [Go SDK Architecture](#go-sdk-architecture)
4. [Installation & Setup](#installation--setup)
5. [Core Services](#core-services)
6. [Agent System](#agent-system)
7. [Advanced Features](#advanced-features)
8. [Functional Options Pattern](#functional-options-pattern)
9. [Error Handling](#error-handling)
10. [Integration Patterns](#integration-patterns)
11. [Best Practices](#best-practices)
12. [Related Ecosystems](#related-ecosystems)
13. [Resources](#resources)

---

## Executive Summary

OpenCode is an **open-source AI coding agent** built for the terminal, designed as an alternative to Claude Code. The OpenCode SDK for Go provides a comprehensive, type-safe REST API client generated with [Stainless](https://www.stainless.com/). It enables programmatic control over OpenCode sessions, file operations, agent management, and terminal UI interactions.

### Key Characteristics

- **Language**: Go with full type safety
- **Pattern**: Functional options pattern for configuration
- **License**: MIT
- **Generation**: Generated with Stainless for maintainability
- **Architecture**: Service-oriented design with 8+ core services
- **Focus**: Production-ready, cloud-native development

---

## OpenCode Ecosystem Overview

### The OpenCode Platform

OpenCode is created by Neovim users and terminal enthusiasts, pushing the limits of what's possible in terminal-based AI development. It serves as an open-source alternative to proprietary solutions, emphasizing:

- **Terminal-first design**: Full TUI (Terminal User Interface) built with Bubble Tea
- **Multi-model support**: OpenAI, Anthropic Claude, Google Gemini, AWS Bedrock, Groq, Azure OpenAI, OpenRouter
- **Multiple AI providers**: Flexible LLM selection and vendor lock-in prevention
- **Session management**: Persistent SQLite storage for conversation history
- **LSP integration**: Language Server Protocol support for deeper IDE integration

### Core Components

1. **OpenCode CLI**: Terminal application providing interactive TUI
2. **OpenCode SDK (Go)**: Programmatic access to OpenCode functionality
3. **OpenCode Server**: Backend service managing sessions, files, and agents
4. **Agent System**: Configurable AI assistants with specialized roles
5. **Tool System**: File operations, code search, bash execution, web fetching

---

## Go SDK Architecture

### Layered Architecture

```
┌─────────────────────────────────────────┐
│     Application Layer                   │
│  (Your Go Application Code)             │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│     SDK Client Layer                    │
│  (opencode.Client with Services)        │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│     Service Layer                       │
│  • Agent    • Session    • File         │
│  • Find     • Config     • App          │
│  • Event    • TUI        • Health       │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│     Options & Middleware Layer          │
│  (Request Options, Functional Options)  │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│     HTTP Transport Layer                │
│  (net/http with automatic retries)      │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│     OpenCode REST API                   │
│  (Server endpoints)                     │
└─────────────────────────────────────────┘
```

### Service Organization

The SDK organizes all functionality into focused services:

| Service | Purpose | Key Use Cases |
|---------|---------|---------------|
| **Agent** | AI agent management and configuration | List agents, configure modes, manage permissions |
| **Session** | Conversation sessions and interactions | Create sessions, send prompts, manage history |
| **File** | File system access and operations | Read files, list directories, check status |
| **Find** | Code search and symbol lookup | Find files, symbols, search text patterns |
| **Config** | OpenCode configuration access | Retrieve provider config, environment info |
| **App** | Application-level operations | Logging, provider info, metadata |
| **Event** | Real-time event streaming | Subscribe to server-sent events |
| **TUI** | Terminal UI control | Show toasts, messages, UI interactions |
| **Health** | Service health and status | Check server availability |

---

## Installation & Setup

### Prerequisites

- Go 1.22 or later
- OpenCode server running locally or accessible via network
- Network connectivity to OpenCode API endpoint

### Basic Installation

```bash
go get -u 'github.com/sst/opencode-sdk-go@v0.19.2'
```

### Standard Client Setup

```go
package main

import (
    "context"
    "github.com/sst/opencode-sdk-go"
)

func main() {
    // Client reads OPENCODE_BASE_URL from environment
    // Falls back to default endpoint if not set
    client := opencode.NewClient()
    
    ctx := context.Background()
    
    // Use the client to interact with OpenCode
    sessions, err := client.Session.List(ctx, opencode.SessionListParams{})
    if err != nil {
        panic(err)
    }
}
```

### Custom Configuration

```go
import (
    "github.com/sst/opencode-sdk-go"
    "github.com/sst/opencode-sdk-go/option"
)

client := opencode.NewClient(
    option.WithBaseURL("http://localhost:8000"),
    option.WithAPIKey("your-api-key"),
    option.WithMaxRetries(3),
)
```

---

## Core Services

### 1. Agent Service

Manage AI agents with specialized configurations, modes, and permissions.

#### Agent Modes

- **`subagent`**: Specialized assistant invoked by primary agents
- **`primary`**: Main interactive agent
- **`all`**: Available in both roles

#### Agent Permissions

Control what actions agents can perform:
- **`bash`**: Execute shell commands
- **`edit`**: Modify files
- **`webfetch`**: Fetch web content

Permission values: `"ask"`, `"allow"`, `"deny"`

#### Usage Examples

```go
// List all agents in a project
agents, err := client.Agent.List(ctx, opencode.AgentListParams{
    Directory: opencode.F("path/to/project"),
})

// Agent structure with full configuration
type Agent struct {
    ID          string
    Mode        opencode.AgentMode        // primary, subagent, all
    Model       string                     // LLM model identifier
    Permissions opencode.AgentPermissions // bash, edit, webfetch settings
    Temperature *float64                   // LLM temperature parameter
    MaxSteps    *int64                     // Iteration limit
    Prompt      string                     // Custom system instructions
}

// Configure agent permissions
type AgentPermissions struct {
    Bash     *string  // "allow", "deny", "ask"
    Edit     *string  // "allow", "deny", "ask"
    Webfetch *string  // "allow", "deny", "ask"
}
```

### 2. Session Service

Create and manage conversation sessions with AI agents.

#### Core Operations

```go
// Create new session
session, err := client.Session.New(ctx, opencode.SessionNewParams{
    Title: opencode.F("Research Task"),
})
sessionID := session.ID

// Send text prompt
response, err := client.Session.Prompt(ctx, sessionID, 
    opencode.SessionPromptParams{
        Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
            opencode.TextPartInputParam{
                Type: opencode.F(opencode.TextPartInputTypeText),
                Text: opencode.F("Analyze this code"),
            },
        }),
    })

// Execute command within session
cmdResult, err := client.Session.Command(ctx, sessionID, 
    opencode.SessionCommandParams{
        Command: opencode.F("ls -la"),
    })

// Run shell command
shellResult, err := client.Session.Shell(ctx, sessionID, 
    opencode.SessionShellParams{
        Command: opencode.F("echo 'Hello'"),
    })

// List all sessions
sessions, err := client.Session.List(ctx, opencode.SessionListParams{})

// Get specific session details
session, err := client.Session.Get(ctx, sessionID)

// Share session
shared, err := client.Session.Share(ctx, sessionID)

// Abort active processing
err = client.Session.Abort(ctx, sessionID)

// Delete session
err = client.Session.Delete(ctx, sessionID)
```

#### Session Message Format

Sessions accept multiple part types:
- **Text**: Plain text prompts
- **Tool Results**: Results from previously executed tools
- **Context**: Document or context files

### 3. File Service

Read and manage files within the OpenCode context.

```go
// List files in directory
files, err := client.File.List(ctx, opencode.FileListParams{
    Path: opencode.F("/"),
})

// Read file content
content, err := client.File.Read(ctx, opencode.FileReadParams{
    Path: opencode.F("main.go"),
})

// Check file status
status, err := client.File.Status(ctx, opencode.FileStatusParams{})

// File list response structure
type FileListResponse struct {
    Files      []FileInfo `json:"files"`
    Directories []string  `json:"directories"`
}

type FileInfo struct {
    Name        string `json:"name"`
    Path        string `json:"path"`
    Size        int64  `json:"size"`
    Modified    string `json:"modified"`
    IsDirectory bool   `json:"is_directory"`
}
```

### 4. Find Service

Search files and code symbols efficiently.

```go
// Find files matching pattern
files, err := client.Find.Files(ctx, opencode.FindFilesParams{
    Query: opencode.F("*.go"),
})

// Find code symbols
symbols, err := client.Find.Symbols(ctx, opencode.FindSymbolsParams{
    Query: opencode.F("MyFunction"),
})

// Find text in files
results, err := client.Find.Text(ctx, opencode.FindTextParams{
    Pattern: opencode.F("searchTerm"),
    Path:    opencode.F("src/"),
})

// Find results structure
type FindResult struct {
    File       string `json:"file"`
    Line       int    `json:"line"`
    Column     int    `json:"column"`
    Text       string `json:"text"`
    Score      float64 `json:"score"` // Relevance score
}

// Symbol result structure
type SymbolResult struct {
    Name       string `json:"name"`
    Type       string `json:"type"`      // function, class, method, etc.
    File       string `json:"file"`
    Line       int    `json:"line"`
    Definition string `json:"definition"`
    Score      float64 `json:"score"`
}
```

### 5. Config Service

Access and retrieve OpenCode configuration.

```go
// Get current configuration
config, err := client.Config.Get(ctx, opencode.ConfigGetParams{})

// Configuration structure
type Config struct {
    BaseURL         string                 `json:"base_url"`
    DefaultAgent    string                 `json:"default_agent"`
    DefaultModel    string                 `json:"default_model"`
    Providers       map[string]Provider    `json:"providers"`
    Agents          map[string]AgentConfig `json:"agents"`
    Environment     map[string]string      `json:"environment"`
}

type Provider struct {
    Type   string            `json:"type"`  // openai, anthropic, gemini, etc.
    Config map[string]string `json:"config"`
}
```

### 6. App Service

Application-level operations and metadata.

```go
// Write log message
logResult, err := client.App.Log(ctx, opencode.AppLogParams{
    Level:   opencode.F(opencode.AppLogParamsLevelInfo),
    Message: opencode.F("Processing started"),
    Service: opencode.F("my-app"),
    Metadata: opencode.F(map[string]string{
        "task_id": "12345",
    }),
})

// Get available providers
providers, err := client.App.Providers(ctx, opencode.AppProvidersParams{})

// Log levels: info, debug, warn, error

// Returns
type LogResult struct {
    ID        string    `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
}
```

### 7. Event Service

Real-time event streaming for reactive applications.

```go
// Stream events from server
stream := client.Event.ListStreaming(ctx, opencode.EventListParams{
    Filter: opencode.F("session.*"),
})

for event := range stream.Events() {
    // Handle event
    println(event.Type, event.Data)
}

// Close stream
stream.Close()

// Event structure
type Event struct {
    ID        string            `json:"id"`
    Type      string            `json:"type"`      // session.created, session.message, etc.
    Timestamp time.Time         `json:"timestamp"`
    SessionID string            `json:"session_id,omitempty"`
    Data      map[string]interface{} `json:"data"`
}
```

### 8. TUI Service

Control terminal UI elements programmatically.

```go
// Show toast notification
toast, err := client.Tui.ShowToast(ctx, opencode.TuiShowToastParams{
    Message: opencode.F("Operation successful"),
    Variant: opencode.F(opencode.TuiShowToastParamsVariantSuccess),
    // Variants: success, error, info, warning
})

// Show confirmation dialog
dialog, err := client.Tui.ShowDialog(ctx, opencode.TuiShowDialogParams{
    Title:   opencode.F("Confirm Action"),
    Message: opencode.F("Do you want to proceed?"),
    Type:    opencode.F(opencode.TuiShowDialogParamsTypeConfirm),
})

// Show input dialog
input, err := client.Tui.ShowInput(ctx, opencode.TuiShowInputParams{
    Label:       opencode.F("Enter value"),
    Placeholder: opencode.F("example value"),
})
```

---

## Agent System

### Agent Architecture

OpenCode features a hierarchical agent system:

#### Primary Agents

Built-in primary agents included with OpenCode:

1. **Build Agent**
   - Default agent with full access
   - Enabled for all development tools (bash, edit, webfetch)
   - Used for active development work
   - Switch to with Tab key

2. **Plan Agent**
   - Read-only analysis agent
   - Restricted permissions (bash and edit disabled)
   - Optimized for code exploration and planning
   - Prevents accidental changes

#### Subagents

Specialized assistants available for specific tasks:

1. **General Subagent**
   - Researches complex questions
   - Executes multi-step tasks
   - Provides in-depth analysis
   - Invoked with `@general` mention

2. **Explore Subagent**
   - Specialized for codebase navigation
   - Optimized for code search
   - Fast symbol and pattern lookup
   - Invoked with `@explore` mention

### Configuration Methods

Agents can be configured in two ways:

#### 1. JSON Configuration (`opencode.json`)

```json
{
  "agents": {
    "custom-agent": {
      "mode": "primary",
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "maxSteps": 20,
      "permissions": {
        "bash": "allow",
        "edit": "ask",
        "webfetch": "allow"
      },
      "prompt": "file:./prompts/custom-agent.txt"
    },
    "analyzer": {
      "mode": "subagent",
      "model": "gpt-4",
      "permissions": {
        "bash": "deny",
        "edit": "deny",
        "webfetch": "allow"
      }
    }
  }
}
```

#### 2. Markdown Configuration (Global or Project-Specific)

**Global location**: `~/.config/opencode/agent/custom-agent.md`  
**Project location**: `.opencode/agent/custom-agent.md`

```markdown
# Custom Agent Configuration

**Mode**: primary
**Model**: claude-3-5-sonnet-20241022
**Temperature**: 0.7

## Permissions

- Bash: allow
- Edit: ask
- Webfetch: allow

## System Prompt

You are a specialized code review agent focusing on...
```

### Key Configuration Options

| Option | Type | Values | Notes |
|--------|------|--------|-------|
| `mode` | String | `primary`, `subagent`, `all` | Determines agent availability |
| `model` | String | Model ID | Override default LLM model |
| `temperature` | Float | 0.0-2.0 | Controls response randomness |
| `maxSteps` | Integer | > 0 | Limits agentic iterations |
| `permissions.bash` | String | `allow`, `deny`, `ask` | Shell command execution |
| `permissions.edit` | String | `allow`, `deny`, `ask` | File modification |
| `permissions.webfetch` | String | `allow`, `deny`, `ask` | HTTP content fetching |
| `prompt` | String | Text or file reference | Custom system instructions |

### Agent Invocation Patterns

```go
// List available agents for project
agents, err := client.Agent.List(ctx, opencode.AgentListParams{
    Directory: opencode.F("./my-project"),
})

// Create session and use specific agent
session, err := client.Session.New(ctx, opencode.SessionNewParams{
    Title:     opencode.F("Analysis Task"),
    AgentID:   opencode.F("analyzer"), // Use specific agent
})

// Mention subagent in prompt
response, err := client.Session.Prompt(ctx, sessionID, 
    opencode.SessionPromptParams{
        Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
            opencode.TextPartInputParam{
                Type: opencode.F(opencode.TextPartInputTypeText),
                Text: opencode.F("@explore Find all handler functions"),
            },
        }),
    })
```

---

## Advanced Features

### 1. Functional Options Pattern

The SDK uses Go's functional options pattern for flexible, composable configuration.

#### Helper Functions

```go
// Generic field helpers
opencode.F(value)           // Generic field wrapper
opencode.String(value)      // String field
opencode.Int(value)         // Int64 field
opencode.Float(value)       // Float64 field
opencode.Bool(value)        // Bool field
opencode.Null[T]()          // Null value
opencode.Raw[T](any)        // Non-conforming type

// Usage example
params := opencode.SessionListParams{
    Limit:  opencode.F(int64(10)),
    Filter: opencode.F("active"),
}
```

#### Request Options

```go
import "github.com/sst/opencode-sdk-go/option"

// Apply options at client level
client := opencode.NewClient(
    option.WithBaseURL("http://localhost:8000"),
    option.WithAPIKey("sk-..."),
    option.WithMaxRetries(5),
    option.WithRequestTimeout(30*time.Second),
)

// Apply options to specific requests
response, err := client.Session.List(ctx, params,
    option.WithHeader("X-Request-ID", "custom-id"),
    option.WithMiddleware(customMiddleware),
)

// Common request options:
// - WithHeader(key, value)
// - WithMaxRetries(n)
// - WithRequestTimeout(duration)
// - WithMiddleware(fn)
// - WithQuery(key, value)
```

### 2. Middleware Support

Register custom middleware for request/response processing.

```go
import (
    "github.com/sst/opencode-sdk-go/option"
)

// Middleware function signature
type Middleware func(*http.Request) (*http.Request, error)

// Logging middleware
loggingMiddleware := func(req *http.Request) (*http.Request, error) {
    println("Request:", req.Method, req.URL)
    return req, nil
}

// Client with middleware
client := opencode.NewClient(
    option.WithMiddleware(loggingMiddleware),
)

// Request-level middleware (runs after client middleware)
response, err := client.Session.List(ctx, params,
    option.WithMiddleware(requestSpecificMiddleware),
)

// Middleware execution order: client → request
```

### 3. Automatic Retries

The SDK includes built-in retry logic:

- **Default**: 2 automatic retries
- **Configurable**: Via `option.WithMaxRetries(n)`
- **Idempotent**: Safe for all read operations and safe mutations
- **Backoff**: Exponential backoff with jitter

```go
// Enable custom retry count
client := opencode.NewClient(
    option.WithMaxRetries(5),
)

// Disable retries
client := opencode.NewClient(
    option.WithMaxRetries(0),
)
```

### 4. Custom HTTP Clients

Substitute the HTTP client for advanced scenarios.

```go
import "net/http"

customHTTPClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

client := opencode.NewClient(
    option.WithHTTPClient(customHTTPClient),
)
```

### 5. Accessing Undocumented Endpoints

For new or internal endpoints not yet in the SDK:

```go
// Generic HTTP methods on client
response, err := client.Get(ctx, "/custom/endpoint", 
    option.WithQuery("param", "value"),
)

response, err := client.Post(ctx, "/custom/endpoint",
    option.WithQuery("param", "value"),
)

// Request options (headers, middleware, etc.) are respected
```

---

## Functional Options Pattern

### Design Philosophy

Go's functional options pattern provides:
- **Type safety**: Full IDE support and compile-time checking
- **Composability**: Stack options for complex configurations
- **Backwards compatibility**: Add options without breaking existing code
- **Clarity**: Intent is explicit in code

### Implementation Details

```go
// RequestOption type - a function that modifies RequestConfig
type RequestOption func(*RequestConfig)

// Helper function that returns a RequestOption
func WithHeader(key, value string) RequestOption {
    return func(config *RequestConfig) {
        config.Headers[key] = value
    }
}

// Usage
response, err := client.Session.List(ctx, params,
    WithHeader("X-Custom", "value"),
    WithMaxRetries(3),
)
```

### Benefits Over Alternatives

| Pattern | Type Safety | Composition | Backwards Compatible |
|---------|------------|-------------|----------------------|
| Functional Options | ✅ Excellent | ✅ Excellent | ✅ Yes |
| Builder Pattern | ✅ Good | ⚠️ Medium | ✅ Yes |
| Struct + Pointers | ❌ Poor | ❌ Limited | ⚠️ Partial |
| Variadic Interfaces | ⚠️ Medium | ⚠️ Medium | ✅ Yes |

---

## Error Handling

### Error Types

```go
import (
    "errors"
    "github.com/sst/opencode-sdk-go"
)

// Custom error type
var apierr *opencode.Error

// Checking for API errors
resp, err := client.Session.List(ctx, params)
if err != nil {
    if errors.As(err, &apierr) {
        // Handle OpenCode API error
        println("Status:", apierr.StatusCode)
        println("Request:", string(apierr.DumpRequest(true)))
        println("Response:", string(apierr.DumpResponse(true)))
    } else {
        // Handle other errors (network, context, etc.)
        panic(err)
    }
}
```

### Error Structure

```go
type Error struct {
    StatusCode int
    Request    *http.Request
    Response   *http.Response
    JSON       map[string]interface{}
}

// Methods
func (e *Error) Error() string                    // Error message
func (e *Error) DumpRequest(verbose bool) []byte // Full request dump
func (e *Error) DumpResponse(verbose bool) []byte // Full response dump
```

### Common HTTP Status Codes

| Code | Meaning | Typical Cause |
|------|---------|---------------|
| 200 | OK | Success |
| 400 | Bad Request | Invalid parameters |
| 401 | Unauthorized | Missing/invalid auth |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Session/agent doesn't exist |
| 429 | Too Many Requests | Rate limited |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | Server offline |

### Error Handling Best Practices

```go
// Comprehensive error handling
resp, err := client.Session.New(ctx, params)
if err != nil {
    var apierr *opencode.Error
    
    if errors.As(err, &apierr) {
        switch apierr.StatusCode {
        case 400:
            // Handle validation error
            log.Printf("Invalid parameters: %v", apierr.JSON)
            return fmt.Errorf("validation failed: %w", err)
        case 401:
            // Handle auth error
            return fmt.Errorf("authentication failed: %w", err)
        case 429:
            // Handle rate limiting
            return fmt.Errorf("rate limited, retry later: %w", err)
        default:
            return fmt.Errorf("api error: %w", err)
        }
    }
    
    // Handle context errors, network errors, etc.
    if errors.Is(err, context.Canceled) {
        return fmt.Errorf("request canceled: %w", err)
    }
    
    return fmt.Errorf("unexpected error: %w", err)
}
```

---

## Integration Patterns

### Pattern 1: Long-Running Session Management

```go
// Create persistent session for multi-turn conversation
session, err := client.Session.New(ctx, opencode.SessionNewParams{
    Title: opencode.F("Code Review Session"),
})
if err != nil {
    log.Fatal(err)
}

sessionID := session.ID

// Reuse session across multiple interactions
for _, codeSnippet := range codeReviewItems {
    response, err := client.Session.Prompt(ctx, sessionID,
        opencode.SessionPromptParams{
            Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
                opencode.TextPartInputParam{
                    Type: opencode.F(opencode.TextPartInputTypeText),
                    Text: opencode.F("Review this code:\n" + codeSnippet),
                },
            }),
        })
    
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }
    
    processResponse(response)
}

// Clean up when done
_ = client.Session.Delete(ctx, sessionID)
```

### Pattern 2: Concurrent Session Operations

```go
import "sync"

// Process multiple tasks concurrently
var wg sync.WaitGroup
sessions := make([]string, 10)

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        
        session, err := client.Session.New(ctx, 
            opencode.SessionNewParams{
                Title: opencode.F(fmt.Sprintf("Task %d", index)),
            })
        
        if err != nil {
            log.Printf("Error creating session: %v", err)
            return
        }
        
        sessions[index] = session.ID
        
        // Process session
        processSession(ctx, client, session.ID)
    }(i)
}

wg.Wait()
```

### Pattern 3: Event-Driven Architecture

```go
// Subscribe to real-time events
stream := client.Event.ListStreaming(ctx, opencode.EventListParams{})

go func() {
    for event := range stream.Events() {
        switch event.Type {
        case "session.created":
            onSessionCreated(event.Data)
        case "session.message":
            onSessionMessage(event.Data)
        case "session.error":
            onSessionError(event.Data)
        }
    }
}()

// Stream runs until closed or context canceled
<-ctx.Done()
stream.Close()
```

### Pattern 4: Agent Task Routing

```go
// Route tasks to appropriate agents
type TaskType string

const (
    ReviewTask     TaskType = "review"
    AnalysisTask   TaskType = "analysis"
    RefactorTask   TaskType = "refactor"
)

func routeTask(ctx context.Context, task TaskType, code string) error {
    var agentID string
    
    switch task {
    case ReviewTask:
        agentID = "code-reviewer"
    case AnalysisTask:
        agentID = "analyzer"
    case RefactorTask:
        agentID = "refactorer"
    }
    
    session, err := client.Session.New(ctx, 
        opencode.SessionNewParams{
            Title:   opencode.F(string(task)),
            AgentID: opencode.F(agentID),
        })
    
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    
    // Send task to routed agent
    _, err = client.Session.Prompt(ctx, session.ID,
        opencode.SessionPromptParams{
            Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
                opencode.TextPartInputParam{
                    Type: opencode.F(opencode.TextPartInputTypeText),
                    Text: opencode.F(code),
                },
            }),
        })
    
    return err
}
```

### Pattern 5: Bulk File Operations

```go
// Efficiently work with multiple files
files, err := client.Find.Files(ctx, opencode.FindFilesParams{
    Query: opencode.F("*.go"),
})
if err != nil {
    return err
}

type FileAnalysis struct {
    Path   string
    Result string
    Error  error
}

results := make(chan FileAnalysis)

for _, file := range files {
    go func(f string) {
        content, err := client.File.Read(ctx, 
            opencode.FileReadParams{
                Path: opencode.F(f),
            })
        
        if err != nil {
            results <- FileAnalysis{Path: f, Error: err}
            return
        }
        
        // Analyze file
        analysis := analyzeCode(content)
        results <- FileAnalysis{Path: f, Result: analysis}
    }(file)
}

// Collect results
for i := 0; i < len(files); i++ {
    result := <-results
    if result.Error != nil {
        log.Printf("Error analyzing %s: %v", result.Path, result.Error)
    } else {
        log.Printf("Analysis of %s: %s", result.Path, result.Result)
    }
}
```

---

## Best Practices

### 1. Context Management

```go
// Always use context with timeout for external operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Propagate context through layers
response, err := client.Session.List(ctx, params)

// For long-running operations, use context.Background() or separate timeout
longCtx := context.Background()
session, err := client.Session.New(longCtx, params)
```

### 2. Error Handling Strategy

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create session: %w", err)
}

// Log errors appropriately
log.Printf("Session error: %v", err)

// Don't swallow errors silently
_ = client.Session.Delete(ctx, sessionID) // Only OK for cleanup operations
```

### 3. Resource Cleanup

```go
// Always clean up sessions
session, _ := client.Session.New(ctx, params)
sessionID := session.ID
defer func() {
    _ = client.Session.Delete(ctx, sessionID) // Ignore cleanup errors
}()

// Close event streams
stream := client.Event.ListStreaming(ctx, params)
defer stream.Close()
```

### 4. Configuration Management

```go
// Load configuration from environment
baseURL := os.Getenv("OPENCODE_BASE_URL")
if baseURL == "" {
    baseURL = "http://localhost:8000"
}

apiKey := os.Getenv("OPENCODE_API_KEY")

client := opencode.NewClient(
    option.WithBaseURL(baseURL),
    option.WithAPIKey(apiKey),
)

// Store client as singleton if reused
var opencodeClient *opencode.Client

func init() {
    opencodeClient = opencode.NewClient()
}
```

### 5. Observability

```go
// Implement logging middleware
loggingMiddleware := func(req *http.Request) (*http.Request, error) {
    log.Printf("[%s] %s %s", req.Method, req.URL.Path, req.Header.Get("Content-Type"))
    return req, nil
}

client := opencode.NewClient(
    option.WithMiddleware(loggingMiddleware),
)

// Log important events
_, _ = client.App.Log(ctx, opencode.AppLogParams{
    Level:   opencode.F(opencode.AppLogParamsLevelInfo),
    Message: opencode.F("Task started"),
    Service: opencode.F("my-app"),
})
```

### 6. Agent Selection

```go
// Check available agents before use
agents, err := client.Agent.List(ctx, opencode.AgentListParams{
    Directory: opencode.F("."),
})
if err != nil {
    log.Fatal(err)
}

agentMap := make(map[string]bool)
for _, agent := range agents {
    agentMap[agent.ID] = true
}

// Verify agent exists before use
if !agentMap["analyzer"] {
    log.Fatal("analyzer agent not found")
}
```

### 7. Pagination and Large Result Sets

```go
// Handle potentially large result sets
var allSessions []opencode.Session
limit := int64(50)
offset := int64(0)

for {
    sessions, err := client.Session.List(ctx, 
        opencode.SessionListParams{
            Limit:  opencode.F(limit),
            Offset: opencode.F(offset),
        })
    
    if err != nil {
        return err
    }
    
    allSessions = append(allSessions, sessions...)
    
    if len(sessions) < int(limit) {
        break
    }
    
    offset += limit
}
```

---

## Related Ecosystems

### Google ADK-Go (Agent Development Kit)

**Description**: Flexible framework for building AI agent systems in Go  
**Focus**: Multi-agent orchestration, cloud-native development  
**Key Features**:
- Code-first agent development
- Multi-agent composition
- Agent-to-Agent (A2A) protocol support
- Gemini optimization (model-agnostic)

**Resources**:
- [GitHub Repository](https://github.com/google/adk-go)
- [Official Documentation](https://google.github.io/adk-docs/)
- [Announcement Blog](https://developers.googleblog.com/en/announcing-the-agent-development-kit-for-go-build-powerful-ai-agents-with-your-favorite-languages/)

### Ingenimax agent-sdk-go

**Description**: Production-ready Go framework for AI agents  
**Focus**: Structured output, YAML configuration, type safety  
**Key Features**:
- YAML-based agent definition
- Automatic JSON schema generation
- Direct struct unmarshaling
- Production-grade error handling

**Resources**:
- [GitHub Repository](https://github.com/Ingenimax/agent-sdk-go)

### Pontus agent-sdk-go

**Description**: AI agent framework inspired by OpenAI's API  
**Focus**: Multiple LLM providers, function calling, agent handoffs  
**Key Features**:
- OpenAI Assistants API compatibility
- Local LLM support
- Function calling and tool use
- Agent-to-agent routing

**Resources**:
- [GitHub Repository](https://github.com/pontus-devoteam/agent-sdk-go)
- [Go Package Documentation](https://pkg.go.dev/github.com/pontus-devoteam/agent-sdk-go)

### OpenAgents Framework

**Description**: Plan-first development workflow with approval-based execution  
**Focus**: Multi-language support, OpenCode integration  
**Key Features**:
- TypeScript, Python, Go, Rust support
- Automatic testing and code review
- Plan-first execution model
- Built for OpenCode

**Resources**:
- [GitHub Repository](https://github.com/darrenhinde/OpenAgents)

---

## Resources

### Official Documentation

- **OpenCode Main Site**: https://opencode.ai/
- **SDK Documentation**: https://opencode.ai/docs/sdk/
- **Agents Documentation**: https://opencode.ai/docs/agents/
- **Introduction Guide**: https://opencode.ai/docs/

### Go SDK Resources

- **GitHub Repository**: https://github.com/sst/opencode-sdk-go
- **Go Package Documentation**: https://pkg.go.dev/github.com/sst/opencode-sdk-go
- **Shared Package**: https://pkg.go.dev/github.com/sst/opencode-sdk-go/shared

### Related Projects

- **OpenCode CLI**: https://github.com/sst/opencode
- **OpenCode (AI Coding Agent)**: https://github.com/opencode-ai/opencode

### Learning Resources

- **Stainless Code Generation**: https://www.stainless.com/
- **Go Best Practices**: https://golang.org/doc/effective_go
- **Go Context Package**: https://golang.org/pkg/context/

### Community

- **OpenCode GitHub Discussions**: https://github.com/sst/opencode/discussions
- **Go SDK Issues**: https://github.com/sst/opencode-sdk-go/issues

---

## Changelog & Versions

| Version | Release Date | Key Changes |
|---------|-------------|-------------|
| v0.19.2 | Dec 2025 | Current stable release |
| v0.19.x | Earlier | Historical versions |

---

## Conclusion

The OpenCode SDK for Go provides a comprehensive, production-ready interface for building AI-powered development tools. By leveraging the functional options pattern, service-oriented architecture, and Stainless-generated code, it offers both flexibility and type safety.

Key takeaways for effective utilization:

1. **Understand the service architecture**: Each service handles a distinct domain (agents, sessions, files, etc.)
2. **Master the functional options pattern**: Use request options for composable, type-safe configuration
3. **Implement proper error handling**: Always check for `opencode.Error` and handle context cancellation
4. **Design for concurrency**: Use goroutines safely with context-based coordination
5. **Leverage agents strategically**: Route tasks to agents with appropriate permissions and capabilities
6. **Keep observability in mind**: Implement logging and monitoring from the start
7. **Stay ecosystem-aware**: Understanding related frameworks (ADK-Go, etc.) provides broader context

The OpenCode ecosystem continues to evolve. Regularly check the official documentation and GitHub repositories for updates and best practices.

---

**Document Version**: 1.0  
**Last Updated**: December 2025  
**Maintained By**: OpenCode Community  
**License**: MIT (same as OpenCode SDK)
