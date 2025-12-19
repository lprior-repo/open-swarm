# OpenCode Platform Architecture - Deep Research & Building Guide

**Last Updated**: December 2025  
**Platform Version**: 0.15.18+  
**Runtime**: Bun (Backend), Go (TUI)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Platform Architecture Overview](#platform-architecture-overview)
3. [Client-Server Architecture](#client-server-architecture)
4. [Web Server & HTTP API](#web-server--http-api)
5. [Plugin System & Extensibility](#plugin-system--extensibility)
6. [Tool Framework](#tool-framework)
7. [MCP Integration](#mcp-integration)
8. [Session Management & State](#session-management--state)
9. [LLM Provider Integration](#llm-provider-integration)
10. [TUI Architecture](#tui-architecture)
11. [Monorepo Structure](#monorepo-structure)
12. [Agent Architecture & Loop](#agent-architecture--loop)
13. [Design Patterns](#design-patterns)
14. [Best Practices for Building](#best-practices-for-building)
15. [Integration Patterns](#integration-patterns)
16. [Deployment & Infrastructure](#deployment--infrastructure)

---

## Executive Summary

OpenCode is a sophisticated, open-source AI coding agent designed for terminal-based development. Unlike monolithic AI tools, it follows a **distributed client-server architecture** that separates the HTTP backend (Bun JavaScript runtime) from the TUI frontend (Go), enabling multiple client interfaces and programmatic access.

### Key Architectural Principles

1. **Separation of Concerns**: HTTP server decoupled from UI layer
2. **Provider Agnosticism**: Support for 75+ LLM providers via AI SDK abstraction
3. **Extensibility**: Plugin system, MCP servers, custom tools
4. **Event-Driven**: Real-time state propagation via SSE and event bus
5. **Graceful Degradation**: Context management, auto-compaction, token tracking
6. **Monorepo Organization**: Modular packages with clear dependencies

### Technology Stack

- **Backend Runtime**: Bun (JavaScript/TypeScript)
- **Server Framework**: Hono (lightweight web framework)
- **TUI Framework**: Bubble Tea (Go)
- **AI SDK**: Vercel AI SDK with provider integrations
- **Package Manager**: Bun workspaces (monorepo)
- **Build Orchestration**: Turbo
- **State Persistence**: File-based with exclusive locks
- **IPC**: SSE (Server-Sent Events), HTTP

---

## Platform Architecture Overview

### High-Level System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Layer                                │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  CLI TUI     │  │  Web Client  │  │  IDE Plugin  │           │
│  │  (Go)        │  │  (Browser)   │  │  (JS/TS)     │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                   │
│           All connect via HTTP + SSE                             │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│               HTTP Server (Bun Runtime)                          │
├────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           OpenCode Core Application                      │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │  Route Handler Layer                             │   │   │
│  │  │  /session, /message, /file, /find, /config, ... │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  │                      ↓                                    │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │  Core Services Layer                             │   │   │
│  │  │  • Agent Service    • Session Service            │   │   │
│  │  │  • Tool Executor    • LLM Controller             │   │   │
│  │  │  • Permission Mgmt  • Event Bus                  │   │   │
│  │  │  • Plugin System    • State Manager              │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  │                      ↓                                    │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │  Integration Layer                               │   │   │
│  │  │  • AI SDK Wrapper   • Tool Framework             │   │   │
│  │  │  • MCP Servers      • LSP Client                 │   │   │
│  │  │  • Plugin Loader    • Event System               │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  │                      ↓                                    │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │  Storage & External Layer                        │   │   │
│  │  │  • File System      • LLM Providers (75+)        │   │   │
│  │  │  • Storage Engine   • MCP Servers                │   │   │
│  │  │  • Git Integration  • External APIs              │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### System Responsibilities

| Layer | Responsibility | Technology |
|-------|-----------------|------------|
| **Clients** | User interaction, rendering, input handling | Go (TUI), HTML/JS (Web), TypeScript (SDK) |
| **HTTP Server** | Request routing, authentication, session management | Bun + Hono |
| **Core Services** | Business logic, agent coordination, tool execution | JavaScript/TypeScript |
| **Integration** | LLM abstraction, plugin loading, event propagation | AI SDK, Custom modules |
| **Storage** | Persistence, file locking, state recovery | File system, In-memory caches |
| **External** | Model access, server integration, environment access | HTTP, SSH, File system |

---

## Client-Server Architecture

### Design Philosophy

OpenCode implements a **server-centric architecture** where:

1. **Single Source of Truth**: HTTP server maintains all session state
2. **Multiple Clients**: Any client can connect to the same server
3. **Stateless Clients**: Clients don't maintain persistent state
4. **Real-Time Sync**: SSE ensures clients always see current state

### Server Invocation Patterns

```bash
# Standalone server (headless)
opencode serve --port 4096 --hostname 127.0.0.1

# TUI automatically starts server (port 4096 by default)
opencode  # Starts both server and TUI client

# Connect existing TUI to running server
opencode --hostname localhost --port 4096
```

### HTTP API Endpoints

The server exposes comprehensive REST API endpoints organized by resource:

#### Session Endpoints
```
POST   /session                   # Create new session
GET    /session                   # List sessions
GET    /session/:id               # Get session details
DELETE /session/:id               # Delete session
POST   /session/:id/fork          # Fork session
POST   /session/:id/abort         # Abort active processing
POST   /session/:id/share         # Enable sharing
DELETE /session/:id/share         # Disable sharing
```

#### Message Endpoints
```
POST   /session/:id/message       # Send message (async)
POST   /session/:id/message/sync  # Send message (sync with response)
POST   /session/:id/command       # Execute command
```

#### File Operations
```
GET    /file/status               # Get file status
POST   /file/find                 # Find files by pattern
GET    /file/read                 # Read file content
```

#### Agent Operations
```
GET    /agent                     # List available agents
GET    /agent/:id                 # Get agent details
```

#### Configuration
```
GET    /config                    # Get configuration
GET    /provider                  # List providers
POST   /provider/auth             # OAuth authorization
```

#### Tools
```
GET    /tool                      # List available tools
GET    /tool/:id/schema           # Get tool JSON schema
```

#### Events (SSE)
```
GET    /global/event              # Stream global events (SSE)
GET    /session/:id/event         # Stream session events (SSE)
```

#### Documentation
```
GET    /doc                       # OpenAPI specification (Swagger UI)
```

### OpenAPI Specification

The server exposes complete OpenAPI 3.1 specification at `/doc` endpoint:

```typescript
// Example: Getting OpenAPI spec
fetch('http://localhost:4096/doc')
  .then(r => r.json())
  .then(spec => {
    // spec contains full API documentation
    // Can be used with code generation tools
  })
```

### Architecture Benefits

**Flexibility**: Run server on one machine, client on another
**Testability**: Test server endpoints independently
**Scaling**: Support multiple concurrent clients
**Programmatic Access**: Script interactions via HTTP
**IDE Integration**: Plugins can interact via REST API

---

## Web Server & HTTP API

### Server Implementation Details

#### Framework: Hono

Hono is a lightweight, edge-computing-friendly web framework:

```typescript
import { Hono } from 'hono'

const app = new Hono()

// Route handlers automatically typed
app.post('/session', async (c) => {
  const body = await c.req.json()
  // Handle request
  return c.json({ id: 'session-123' })
})

app.get('/session/:id', async (c) => {
  const id = c.req.param('id')
  // Query and return session
  return c.json({ id, title: '...' })
})
```

#### Middleware Support

```typescript
// Authentication middleware
app.use('*', async (c, next) => {
  const token = c.req.header('Authorization')
  if (!token) return c.text('Unauthorized', 401)
  // Verify token
  await next()
})

// Logging middleware
app.use('*', async (c, next) => {
  const start = Date.now()
  await next()
  const duration = Date.now() - start
  console.log(`${c.req.method} ${c.req.path} - ${duration}ms`)
})

// CORS
app.use('*', cors())
```

### HTTP Pattern: REST with SSE

The API follows REST conventions for CRUD but uses SSE for real-time updates:

#### Pattern: Async Operations with Streaming

```typescript
// Client creates session
const sessionResp = await fetch('/session', {
  method: 'POST',
  body: JSON.stringify({ title: 'My Task' })
})
const session = await sessionResp.json()

// Client subscribes to events
const eventSource = new EventSource(`/session/${session.id}/event`)

eventSource.addEventListener('message.added', (e) => {
  const message = JSON.parse(e.data)
  // Update UI with new message
})

// Client sends prompt (async)
await fetch(`/session/${session.id}/message`, {
  method: 'POST',
  body: JSON.stringify({ text: 'Review this code' })
})

// Agent processes request asynchronously
// Events stream back via SSE
// Client updates UI in real-time
```

### Response Format

#### Success Response
```json
{
  "id": "session-123",
  "title": "Code Review",
  "createdAt": "2025-12-19T10:00:00Z",
  "messages": [
    {
      "id": "msg-1",
      "role": "user",
      "content": "Review this function",
      "parts": [ /* ... */ ]
    }
  ]
}
```

#### Error Response
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Session not found",
    "details": {
      "sessionId": "nonexistent"
    }
  }
}
```

### Built-in Route Handlers

Located in `packages/opencode/src/route/`:

```
route/
├── agent.ts          # Agent endpoints
├── session.ts        # Session endpoints
├── message.ts        # Message endpoints
├── file.ts          # File operations
├── config.ts        # Configuration
├── provider.ts      # Provider management
├── tool.ts          # Tool listing
├── event.ts         # Event streaming
└── index.ts         # Route registration
```

### Server Startup Process

```typescript
// cmd/root.ts - Server initialization
async function startServer() {
  // 1. Load configuration
  const config = await Config.load()
  
  // 2. Initialize storage
  const storage = new Storage(config.dataDir)
  
  // 3. Initialize plugin system
  const plugins = await loadPlugins(config)
  
  // 4. Initialize event bus
  const eventBus = new EventBus()
  
  // 5. Create route handlers with context
  const handlers = createRouteHandlers({
    storage,
    plugins,
    eventBus,
    config
  })
  
  // 6. Setup Hono server
  const app = createHonoApp(handlers)
  
  // 7. Start listening
  app.listen({
    port: config.port,
    hostname: config.hostname
  })
}
```

---

## Plugin System & Extensibility

### Plugin Architecture

Plugins are the primary extensibility mechanism for OpenCode. They allow:

- Custom tool creation
- Event-driven automation
- Session lifecycle hooks
- Environmental customization
- External service integration

### Plugin Structure

```typescript
// ~/.config/opencode/plugin/custom-plugin.ts
import { Plugin } from '@opencode-ai/plugin'

export default {
  name: 'custom-plugin',
  description: 'Custom tool for specialized tasks',
  
  // Lifecycle hooks
  async onSessionCreated(ctx, session) {
    // Handle new session
    ctx.client.log('Session created: ' + session.id)
  },
  
  // Event hooks
  async onToolExecuted(ctx, tool) {
    // Log tool execution
    ctx.client.log('Executed: ' + tool.name)
  },
  
  // Tool definitions
  tools: {
    custom: {
      name: 'custom-tool',
      description: 'A custom tool',
      schema: {
        type: 'object',
        properties: {
          input: { type: 'string' }
        },
        required: ['input']
      },
      execute: async (input) => {
        // Tool implementation
        return 'Result: ' + input
      }
    }
  }
} as Plugin
```

### Plugin Context

Plugins receive a context object with access to:

```typescript
interface PluginContext {
  // Project information
  project: {
    name: string
    path: string
    worktree: string
  }
  
  // OpenCode SDK client for programmatic access
  client: OpencodeClient
  
  // Bun shell API for command execution
  $: typeof shell
  
  // Current working directory
  directory: string
}
```

### Plugin Hook Types

Plugins can hook into lifecycle events organized by category:

#### Command Events
- `command.executed` - After command execution

#### File Events
- `file.edited` - When file is modified
- `file.watcher.updated` - When watched files change

#### Installation Events
- `installation.updated` - When installation changes

#### LSP Events
- `lsp.diagnostics.updated` - When diagnostics change
- `lsp.server.updated` - When LSP server state changes

#### Message Events
- `message.part.created` - New message part
- `message.part.updated` - Message part updated
- `message.created` - Full message created
- `message.finished` - Message processing complete

#### Permission Events
- `permission.replied` - Permission decision made
- `permission.updated` - Permission settings changed

#### Server Events
- `server.connected` - Server connection established

#### Session Events
- `session.created` - New session created
- `session.compacted` - Session auto-compacted
- `session.deleted` - Session removed
- `session.status.updated` - Status changed
- `session.error` - Error occurred

#### Tool Events
- `tool.before` - Before tool execution
- `tool.after` - After tool execution

#### TUI Events
- `tui.prompt.submitted` - User submitted prompt
- `tui.command.executed` - Command executed
- `tui.notification.sent` - Notification displayed

### Plugin Configuration

Plugins are configured with conditional initialization:

```json
{
  "disabled_hooks": [
    "file.watcher.updated",
    "tool.before"
  ],
  "plugins": [
    "custom-plugin",
    "another-plugin"
  ]
}
```

### Plugin Loading & Execution

```typescript
// Plugin discovery and loading
async function loadPlugins(config: Config) {
  const locations = [
    `~/.config/opencode/plugin`,  // Global
    `.opencode/plugin`             // Project-specific
  ]
  
  const plugins = []
  for (const location of locations) {
    const files = await glob(`${location}/**/*.{ts,js}`)
    for (const file of files) {
      const module = await import(file)
      plugins.push(module.default)
    }
  }
  
  return plugins
}

// Plugin execution
async function executePluginHook(
  plugins: Plugin[],
  hook: string,
  context: PluginContext,
  ...args: any[]
) {
  for (const plugin of plugins) {
    if (!plugin[hook]) continue
    
    try {
      await plugin[hook](context, ...args)
    } catch (err) {
      console.error(`Plugin error in ${hook}:`, err)
    }
  }
}
```

### Plugin Best Practices

1. **Minimal Dependencies**: Keep plugins lightweight
2. **Error Handling**: Never crash the main process
3. **Async Operations**: Support long-running tasks
4. **Event Cleanup**: Remove listeners in cleanup hook
5. **Type Safety**: Use TypeScript for better IDE support
6. **Scope Awareness**: Know whether hook is project or global
7. **Documentation**: Clear purpose and configuration

### Advanced Plugin Pattern: Custom Tools

```typescript
export default {
  name: 'code-review-plugin',
  
  tools: {
    review: {
      name: 'code-review',
      description: 'Review code with custom rules',
      schema: {
        type: 'object',
        properties: {
          code: { type: 'string', description: 'Code to review' },
          language: { type: 'string', description: 'Programming language' }
        },
        required: ['code', 'language']
      },
      execute: async ({ code, language }) => {
        // Call custom linter
        const issues = await runCustomLinter(code, language)
        return {
          issues,
          summary: `Found ${issues.length} issues`
        }
      }
    }
  },
  
  async onToolExecuted(ctx, result) {
    if (result.tool.name === 'code-review') {
      // Notify user
      ctx.client.notification('Code review complete')
    }
  }
} as Plugin
```

---

## Tool Framework

### Architecture Overview

The tool framework provides a standardized interface for agent capabilities:

```
┌─────────────────────────────────┐
│     LLM (Claude, GPT-4, etc)    │
└──────────────┬──────────────────┘
               │
               ↓ Tool Calls
┌─────────────────────────────────┐
│     Tool Resolution Layer        │
│  (Match tool name to handler)    │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│    Tool Validation Layer        │
│  (Validate inputs against schema)│
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│   Permission Layer              │
│  (Check user permissions)        │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│  Tool Execution Layer           │
│  (Execute tool, capture result) │
└──────────────┬──────────────────┘
               │
               ↓ Tool Result
┌─────────────────────────────────┐
│     LLM (Continue reasoning)    │
└─────────────────────────────────┘
```

### Built-in Tool Categories

#### File Manipulation
- **read**: Read file content with line ranges
- **write**: Create/overwrite files
- **edit**: Modify files with exact string replacements
- **patch**: Apply patch files

#### Code Search
- **grep**: Regex-based search with ripgrep
- **glob**: Find files by pattern
- **list**: Directory listing with filters

#### Execution
- **bash**: Execute shell commands
- **patch**: Apply patch files

#### Web Access
- **webfetch**: Fetch HTTP content
  - Built-in HTML-to-text conversion
  - Markdown conversion support
  - 5MB response limit
  - 30-second timeout (configurable)

#### Task Delegation
- **task**: Invoke subagents for specialized work
- **todowrite**: Create/manage task lists
- **todoread**: Read task list state

### Tool Definition Schema

```typescript
interface Tool {
  name: string
  description: string
  
  schema: {
    type: 'object'
    properties: Record<string, JSONSchema>
    required: string[]
  }
  
  execute: (input: Record<string, any>) => Promise<string>
  
  // Optional
  permissions?: {
    required?: 'bash' | 'edit' | 'webfetch'
    requiresApproval?: boolean
  }
}
```

### Tool Configuration

#### Global Configuration
```json
{
  "tools": {
    "bash": false,
    "edit": "ask",
    "webfetch": true
  }
}
```

#### Per-Agent Configuration
```json
{
  "agents": {
    "plan": {
      "tools": {
        "bash": "ask",
        "edit": "deny",
        "webfetch": true
      }
    }
  }
}
```

#### Wildcard Configuration
```json
{
  "tools": {
    "mymcp_*": true,        // Enable all tools from mymcp server
    "github_*": "ask"       // Ask for approval for github tools
  }
}
```

### Tool Execution Lifecycle

```typescript
async function executeTool(
  toolName: string,
  input: Record<string, any>
) {
  // 1. Find tool
  const tool = toolRegistry.get(toolName)
  if (!tool) throw new Error('Tool not found')
  
  // 2. Validate input
  const validation = validateInput(tool.schema, input)
  if (!validation.valid) {
    throw new Error('Invalid input: ' + validation.errors)
  }
  
  // 3. Check permissions
  if (tool.permissions?.required) {
    const permitted = await checkPermission(
      tool.permissions.required,
      input
    )
    if (!permitted) {
      throw new Error('Permission denied')
    }
  }
  
  // 4. Execute
  const result = await tool.execute(input)
  
  // 5. Emit event
  eventBus.emit('tool.executed', {
    tool: toolName,
    input,
    result
  })
  
  return result
}
```

### Permission System

Three permission levels:

```typescript
type Permission = 'allow' | 'deny' | 'ask'

// 'allow': Always permit without asking
// 'deny': Always reject
// 'ask': Prompt user for decision
```

#### Permission Checking

```typescript
async function checkPermission(
  tool: string,
  action: 'bash' | 'edit' | 'webfetch',
  input: Record<string, any>
): Promise<boolean> {
  const permission = getToolPermission(tool, action)
  
  switch (permission) {
    case 'allow':
      return true
    
    case 'deny':
      return false
    
    case 'ask':
      // Show user dialog
      return await promptUser(
        `Allow tool '${tool}' to ${action}?`,
        input
      )
  }
}
```

#### Permission Safety

The bash tool implements sophisticated permission checking:

1. Parse command using tree-sitter
2. Extract command name and arguments
3. Match against wildcard patterns
4. Check permission for matched command

Common patterns:
- `ls*` - Allow ls, ls -l, ls -la, etc.
- `rm` - Deny (destructive)
- `node script.js` - Specific command

### WebFetch Tool Details

```typescript
interface WebFetchOptions {
  timeout?: number        // Default: 30s, max: 2m
  format?: 'text' | 'markdown' | 'html'
  maxSize?: number       // Default: 5MB
  headers?: Record<string, string>
}

// Implementation
async function webfetch(url: string, options?: WebFetchOptions) {
  // 1. Validate URL
  const parsed = new URL(url)
  
  // 2. Fetch with timeout
  const response = await fetch(url, {
    timeout: options?.timeout || 30000,
    headers: options?.headers
  })
  
  // 3. Check content-type negotiation
  const contentType = response.headers.get('content-type')
  const format = options?.format || negotiateFormat(contentType)
  
  // 4. Stream and convert
  const html = await response.text()
  
  if (format === 'text') {
    // Use HTMLRewriter to extract text
    return extractText(html)
  } else if (format === 'markdown') {
    // Convert to markdown using TurndownService
    return htmlToMarkdown(html)
  }
  
  return html
}
```

### Custom Tool Pattern

```typescript
// Define custom tool
const myTool = {
  name: 'analyze-metrics',
  description: 'Analyze performance metrics',
  
  schema: {
    type: 'object',
    properties: {
      filePath: { type: 'string', description: 'Path to metrics file' },
      threshold: { type: 'number', description: 'Alert threshold' }
    },
    required: ['filePath']
  },
  
  execute: async ({ filePath, threshold = 100 }) => {
    const data = await readMetricsFile(filePath)
    const analyzed = analyzeMetrics(data, threshold)
    
    return JSON.stringify(analyzed, null, 2)
  }
}

// Register in plugin
export default {
  name: 'metrics-plugin',
  tools: {
    metrics: myTool
  }
} as Plugin
```

---

## MCP Integration

### Model Context Protocol Overview

MCP (Model Context Protocol) is a standardized protocol for LLMs to access resources and tools from external servers.

### Integration Architecture

```
┌─────────────────────────────┐
│   OpenCode Agent/LLM        │
└────────────┬────────────────┘
             │
             ↓ Tool Call
┌─────────────────────────────┐
│  MCP Client (OpenCode)      │
└────────────┬────────────────┘
             │
             ↓ MCP Protocol
┌─────────────────────────────┐
│  MCP Server                 │
│  (Local or Remote)          │
└─────────────────────────────┘
             │
             ↓ Resource/Tool Call
┌─────────────────────────────┐
│  External Service           │
│  (GitHub, Database, etc)    │
└─────────────────────────────┘
```

### Configuration Structure

```json
{
  "mcp": {
    "servers": {
      "github": {
        "type": "local",
        "command": "node",
        "args": ["./mcp-github-server.js"]
      },
      "context": {
        "type": "remote",
        "url": "https://api.example.com/mcp"
      },
      "database": {
        "type": "local",
        "command": "python",
        "args": ["mcp_db_server.py"],
        "env": {
          "DB_CONNECTION": "postgresql://..."
        }
      }
    }
  }
}
```

### MCP Server Types

#### Local Servers (Started by OpenCode)
```json
{
  "type": "local",
  "command": "node",
  "args": ["./server.js"],
  "env": {
    "DEBUG": "true"
  }
}
```

OpenCode:
1. Spawns process with specified command
2. Communicates via stdin/stdout
3. Manages process lifecycle
4. Restarts on failure

#### Remote Servers (HTTP)
```json
{
  "type": "remote",
  "url": "https://api.example.com/mcp",
  "headers": {
    "Authorization": "Bearer token"
  }
}
```

OpenCode:
1. Makes HTTP requests to URL
2. Authenticates with provided headers
3. Supports OAuth via Dynamic Client Registration

### Tool Integration

MCP server tools are automatically integrated:

```typescript
// MCP server tools appear in OpenCode tool registry
const tools = await client.tool.list()

// Example response
[
  {
    name: 'github_search_repos',
    description: 'Search GitHub repositories',
    schema: { /* JSON schema */ }
  },
  {
    name: 'github_create_issue',
    description: 'Create GitHub issue',
    schema: { /* JSON schema */ }
  }
]

// Tools can be called like built-in tools
const result = await executeTool('github_search_repos', {
  query: 'ai agents',
  language: 'go'
})
```

### Configuration Best Practices

#### Context Management
```json
{
  "mcp": {
    "servers": {
      "expensive": {
        "type": "remote",
        "url": "https://api.example.com/mcp"
      }
    },
    "disabled": ["expensive"]
  }
}
```

**Warning**: MCP servers add to context window. Some like GitHub's implementation consume significant tokens.

#### Tool Filtering
```json
{
  "tools": {
    "github_*": "ask",
    "github_read_*": true,
    "github_write_*": "deny"
  }
}
```

#### OAuth Configuration
```json
{
  "mcp": {
    "servers": {
      "authenticated": {
        "type": "remote",
        "url": "https://api.example.com/mcp",
        "oauth": true
      }
    }
  }
}
```

OpenCode handles OAuth via Dynamic Client Registration (RFC 7591):
1. Server returns OAuth endpoints
2. OpenCode registers as client
3. User authenticates in browser
4. Token managed automatically

### Popular MCP Servers

| Server | Type | Use Case |
|--------|------|----------|
| **GitHub** | Local/Remote | Repository access, issues, PRs |
| **Context7** | Remote | Documentation search |
| **Grep by Vercel** | Remote | GitHub code search |
| **Bright Data** | Remote | Web scraping, proxy |
| **Custom DB** | Local | Database queries |

---

## Session Management & State

### Session Architecture

Sessions are the core unit of work in OpenCode. Each session represents a conversation thread with an AI agent.

### Session Lifecycle

```
┌─────────────┐
│   Created   │
└──────┬──────┘
       │
       ↓
┌─────────────────────┐
│  Active Processing  │ ← User sends prompt/command
├─────────────────────┤
│ • LLM generates     │
│ • Tools execute     │
│ • Events emitted    │
└──────┬──────────────┘
       │
       ├─────→ Compacted (if context > 95% limit)
       │
       ├─────→ Archived (after inactivity)
       │
       ├─────→ Forked (user creates branch)
       │
       └─────→ Deleted
```

### Session Data Structure

```typescript
interface Session {
  // Identification
  id: string              // Unique session ID
  projectId: string       // Project containing session
  
  // Metadata
  title: string           // User-defined title
  createdAt: Date         // Creation timestamp
  updatedAt: Date         // Last activity timestamp
  
  // State
  status: 'idle' | 'running' | 'paused' | 'error'
  
  // Content
  messages: Message[]     // All messages in session
  messageCount: number    // Total message count
  
  // Optimization
  summaryMessageId?: string  // Message containing summarized context
  
  // Cost Tracking
  totalTokens: number     // Total tokens used
  totalCost: number       // Total cost in dollars
  
  // Sharing
  shareToken?: string     // Token for session sharing
  isShared: boolean       // Whether publicly shared
}

interface Message {
  id: string
  sessionId: string
  role: 'user' | 'assistant'
  createdAt: Date
  
  // Message parts (multi-part messages)
  parts: MessagePart[]
}

type MessagePart = 
  | { type: 'text', text: string }
  | { type: 'tool_call', toolName: string, input: Record<string, any> }
  | { type: 'tool_result', toolName: string, result: string }
  | { type: 'error', error: string }
```

### State Persistence Strategy

OpenCode uses a **hierarchical file-based storage system** with exclusive write locks:

```
~/.opencode/
├── sessions/
│   ├── {projectID}/
│   │   ├── {sessionID}/
│   │   │   ├── info.json        # Session metadata
│   │   │   ├── messages.jsonl   # Message log (append-only)
│   │   │   ├── parts.jsonl      # Message parts
│   │   │   └── diffs.jsonl      # File diffs
│   │   └── {sessionID}/
│   │       └── ...
│   └── {projectID}/
│       └── ...
└── storage.db                    # SQLite for indexes
```

### Storage Implementation

```typescript
// Exclusive file locking prevents corruption
async function persistSession(session: Session) {
  const path = `${sessionDir}/${session.id}/info.json`
  
  // Acquire exclusive lock
  const lock = await fs.lock(path)
  
  try {
    // Write atomically
    const tmp = path + '.tmp'
    await fs.writeFile(tmp, JSON.stringify(session))
    await fs.rename(tmp, path)
  } finally {
    // Always release lock
    await lock.release()
  }
}

// Hierarchical storage for scalability
interface StorageKey {
  namespace: 'session' | 'message' | 'part' | 'share'
  projectId: string
  sessionId: string
  messageId?: string
  partId?: string
}

// Storage.read(['message', projectId, sessionId, messageId])
// Storage.write(['message', projectId, sessionId, messageId], data)
```

### Real-Time State Management

Sessions integrate with the **global event bus** for real-time updates:

```typescript
// Event flow
User Input
  ↓
Session Process
  ↓
Event Emission (Tool Executed, Message Added, etc)
  ↓
Event Bus
  ↓
├─→ Persisted to disk
├─→ Broadcast via SSE
└─→ Plugin hooks triggered
```

### Session Status Tracking

Separate from persisted session, status is tracked in-memory:

```typescript
interface SessionStatus {
  sessionId: string
  status: 'idle' | 'processing' | 'waiting_input'
  
  // Current operation
  currentTool?: string
  currentToolProgress?: number
  
  // Metrics
  startTime: Date
  lastActivityTime: Date
  messagesSinceLastCompaction: number
  
  // Cost tracking
  currentTokens: number
  currentCost: number
}

// Status updates published via SSE
eventBus.emit('session.status.updated', {
  sessionId,
  status: 'processing',
  currentTool: 'bash'
})
```

### Auto-Compaction Feature

When sessions approach context limits, OpenCode automatically summarizes:

```typescript
async function maybeCompactSession(session: Session) {
  const tokenUsage = calculateTokenUsage(session)
  const contextLimit = session.model.contextWindow
  
  // Trigger at 95% of context limit
  if (tokenUsage > contextLimit * 0.95) {
    // Generate summary of all prior messages
    const summary = await generateSummary(session.messages)
    
    // Create summary message
    const summaryMessage = {
      id: uuid(),
      role: 'assistant',
      parts: [{ type: 'text', text: summary }]
    }
    
    // Remove old messages, keep summary
    session.messages = [summaryMessage, ...session.messages.slice(-10)]
    session.summaryMessageId = summaryMessage.id
    
    // Persist compacted session
    await persistSession(session)
    
    // Emit event
    eventBus.emit('session.compacted', { sessionId: session.id })
  }
}
```

### Session Forking

Users can branch sessions at any point:

```typescript
async function forkSession(
  sourceSessionId: string,
  branch: {
    title: string
    fromMessageId?: string
  }
): Promise<Session> {
  // Load source session
  const source = await loadSession(sourceSessionId)
  
  // Create new session
  const forked = {
    ...source,
    id: uuid(),
    title: branch.title,
    createdAt: new Date(),
    updatedAt: new Date(),
    
    // Truncate messages if specified
    messages: fromMessageId 
      ? source.messages.slice(0, findIndex(fromMessageId) + 1)
      : source.messages
  }
  
  // Persist as new session
  await persistSession(forked)
  
  return forked
}
```

### Session Lifecycle in Practice

```typescript
// 1. Create session
const session = await client.session.new({
  title: 'Code Review'
})

// 2. Subscribe to events
const events = client.event.stream(`/session/${session.id}/event`)

// 3. Send prompt
await client.session.prompt(session.id, {
  text: 'Review this function'
})

// Real-time events:
// - message.added (user message)
// - tool.before (tool about to run)
// - tool.executed (tool completed)
// - message.added (assistant response)
// - session.status.updated (status changed)

// 4. Continue conversation
await client.session.prompt(session.id, {
  text: '@explore Find similar functions'
})

// 5. Fork session
const forked = await client.session.fork(session.id, {
  title: 'Alternative Approach'
})

// 6. Delete when done
await client.session.delete(session.id)
```

---

## LLM Provider Integration

### Provider Agnosticity Philosophy

OpenCode abstracts away provider-specific details through the **Vercel AI SDK**:

```
Your Code
  ↓
OpenCode AI SDK Wrapper
  ↓
Vercel AI SDK
  ↓
Provider-Specific Package (@ai-sdk/anthropic, @ai-sdk/openai, etc)
  ↓
LLM Provider API
```

### Supported Providers

| Provider | Package | Models | Notes |
|----------|---------|--------|-------|
| **Anthropic** | `@ai-sdk/anthropic` | Claude 3.5 Sonnet, Opus, etc | Default |
| **OpenAI** | `@ai-sdk/openai` | GPT-4, GPT-4o, etc | Popular |
| **Google** | `@ai-sdk/google` | Gemini 2, Gemini Pro | Fast |
| **AWS Bedrock** | `@ai-sdk/aws-bedrock` | Claude, Llama, Titan | Private |
| **Groq** | `@ai-sdk/groq` | Mixtral, Llama | Ultra-fast |
| **Azure OpenAI** | `@ai-sdk/azure-openai` | GPT-4 | Enterprise |
| **OpenRouter** | `@ai-sdk/openrouter` | 75+ models | Multi-provider |
| **Ollama** | `@ai-sdk/openai-compatible` | Local models | Self-hosted |
| **vLLM** | `@ai-sdk/openai-compatible` | Custom models | Self-hosted |

### Provider Configuration

```json
{
  "provider": {
    "default": "anthropic",
    
    "anthropic": {
      "apiKey": "sk-ant-...",
      "baseURL": "https://api.anthropic.com"
    },
    
    "openai": {
      "apiKey": "sk-..."
    },
    
    "local": {
      "type": "openai-compatible",
      "baseURL": "http://localhost:8000",
      "apiKey": "not-needed"
    }
  },
  
  "models": {
    "default": "claude-3-5-sonnet-20241022",
    "fast": "gpt-4o-mini",
    "powerful": "claude-3-opus-20250219"
  }
}
```

### Provider SDK Loading

OpenCode dynamically loads provider SDKs on-demand:

```typescript
// Provider.getSDK() - Loads and caches provider SDK
async function loadProviderSDK(
  provider: string,
  options: Record<string, any>
) {
  // Compute cache key from provider + options hash
  const cacheKey = `${provider}:${hash(options)}`
  
  // Return cached if available
  if (sdkCache.has(cacheKey)) {
    return sdkCache.get(cacheKey)
  }
  
  // Install provider package if not present
  const pkg = `@ai-sdk/${provider}`
  if (!isInstalled(pkg)) {
    await BunProc.install(pkg)
  }
  
  // Import and instantiate
  const module = await import(pkg)
  const sdk = module.default(options)
  
  // Cache and return
  sdkCache.set(cacheKey, sdk)
  return sdk
}
```

### Model Loading

```typescript
// Provider.getModel() - Get model instance
async function getModel(modelId: string) {
  // Parse provider from model ID or use default
  const [provider, modelName] = parseModelId(modelId)
  
  // Load SDK
  const sdk = await loadProviderSDK(provider)
  
  // Get model instance
  const model = sdk.getModel(modelName)
  
  // Wrap with cost tracking
  return wrapModelWithCostTracking(model, modelId)
}
```

### Agent Loop Integration

```typescript
async function runAgentLoop(
  session: Session,
  userMessage: string
) {
  // 1. Get configured model
  const model = await getModel(session.model)
  
  // 2. Build system prompt
  const systemPrompt = buildSystemPrompt(session.agent)
  
  // 3. Collect available tools
  const tools = await getSessionTools(session)
  
  // 4. Stream text with tools
  const { textStream, toolResults } = await streamText({
    model,
    system: systemPrompt,
    messages: session.messages,
    tools: tools,
    
    // Control agentic loop
    stopWhen: (result) => {
      // Stop if max steps reached
      if (result.steps >= session.maxSteps) return true
      
      // Stop if model says done
      if (result.finishReason === 'stop') return true
      
      return false
    }
  })
  
  // 5. Persist results
  for await (const part of textStream) {
    // Handle different part types
    handlePart(part)
  }
}
```

### Cost Tracking

```typescript
interface CostTracking {
  model: string
  inputTokens: number
  outputTokens: number
  
  // Per-token pricing
  inputCost: number     // inputTokens * rate
  outputCost: number    // outputTokens * rate
  totalCost: number
}

async function trackModelCost(
  model: string,
  usage: { inputTokens: number; outputTokens: number }
) {
  const pricing = await getPricing(model)
  
  const cost: CostTracking = {
    model,
    inputTokens: usage.inputTokens,
    outputTokens: usage.outputTokens,
    inputCost: usage.inputTokens * pricing.inputRate,
    outputCost: usage.outputTokens * pricing.outputRate,
    totalCost: 
      (usage.inputTokens * pricing.inputRate) +
      (usage.outputTokens * pricing.outputRate)
  }
  
  // Update session tracking
  session.totalTokens += usage.inputTokens + usage.outputTokens
  session.totalCost += cost.totalCost
  
  // Log for user awareness
  console.log(`Cost: $${cost.totalCost.toFixed(4)}`)
}
```

### Local Model Support

```json
{
  "provider": {
    "local_ollama": {
      "type": "openai-compatible",
      "baseURL": "http://localhost:11434/v1",
      "apiKey": ""
    }
  },
  
  "models": {
    "local": "ollama:mistral"
  }
}
```

When using local models:
- No API keys required
- Models run on your hardware
- Infinite "rate limit"
- Full data privacy
- Slower than cloud (usually)

---

## TUI Architecture

### Bubble Tea Framework

The TUI uses Bubble Tea, a Go framework based on the Elm Architecture:

```
┌──────────────────────────────────────────────────┐
│  Model (State)                                   │
│  - Session data                                  │
│  - UI component state                           │
│  - Input buffers                                │
└──────────────┬───────────────────────────────────┘
               │
        ┌──────┴───────┐
        ↓              ↓
   ┌─────────┐   ┌─────────┐
   │ Events  │   │Messages │
   └─────────┘   └─────────┘
        │              │
        └──────┬───────┘
               ↓
    ┌────────────────────────────┐
    │ Update(model, msg) → model │
    │ (Pure function)            │
    └────────────────────────────┘
               │
               ↓
    ┌────────────────────────────┐
    │ View(model) → string       │
    │ (Render to terminal)       │
    └────────────────────────────┘
```

### Core TUI Components

```typescript
// appModel - Central orchestrator
interface appModel struct {
  // Pages
  pages: map[PageID]Page
  currentPage: PageID
  
  // Dialogs
  sessionDialog: SessionDialog
  confirmDialog: ConfirmDialog
  inputDialog: InputDialog
  
  // Global state
  sessions: []Session
  currentSession?: Session
  events: EventBus
  
  // Focus
  focusedComponent: ComponentID
}

// Implement tea.Model interface
impl appModel {
  fn Init() tea.Cmd {
    // Initialize subscriptions, fetch data
    return tea.Batch(
      subscribeToEvents(),
      loadSessions()
    )
  }
  
  fn Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Route message to focused component
    // Execute commands
  }
  
  fn View() string {
    // Render current page + dialogs
    // Return terminal-ready string
  }
}
```

### Page-Based Navigation

```typescript
type PageID = 
  | 'sessions'
  | 'chat'
  | 'files'
  | 'settings'
  | 'help'

interface Page {
  id: PageID
  title: string
  
  init(): void
  update(msg: tea.Msg): tea.Msg[]
  view(): string
  
  onEnter(): void
  onExit(): void
}

// Switch pages
function switchPage(pageID: PageID) {
  currentPage.onExit()
  currentPage = pages[pageID]
  currentPage.onEnter()
}
```

### Event System

```typescript
// Event subscription from server
function setupSubscriptions() {
  const eventSource = new EventSource('/global/event')
  
  eventSource.addEventListener('session.created', (e) => {
    const event = JSON.parse(e.data)
    sendToTUI(SessionCreatedMsg(event))
  })
  
  eventSource.addEventListener('message.added', (e) => {
    const event = JSON.parse(e.data)
    sendToTUI(MessageAddedMsg(event))
  })
  
  eventSource.addEventListener('tool.executed', (e) => {
    const event = JSON.parse(e.data)
    sendToTUI(ToolExecutedMsg(event))
  })
}
```

### Rendering Pipeline

```
Model State
    ↓
Component Tree
    ├─ Header
    ├─ MainView
    │  ├─ SessionPanel
    │  └─ ChatPanel
    ├─ Footer
    └─ Modals
    ↓
Lip Gloss Styling
    ├─ Colors
    ├─ Borders
    ├─ Padding
    └─ Width constraints
    ↓
Terminal Escape Sequences
    ↓
Rendered String
```

### Key Binding System

```typescript
// Vim-like keybindings
const keybindings = {
  'j': () => move('down'),
  'k': () => move('up'),
  'h': () => switchPage('files'),
  'l': () => switchPage('chat'),
  'i': () => enterInsertMode(),
  'Esc': () => exitInsertMode(),
  'ctrl+c': () => quit(),
  'ctrl+s': () => saveSession()
}

fn handleKeypress(key: string) {
  if (let handler = keybindings[key]) {
    return handler()
  }
}
```

### Component Composition

```typescript
// Base component interface
interface Component {
  init(): void
  update(msg: tea.Msg): tea.Cmd[]
  view(): string
}

// Composite pattern
class ChatPanel implements Component {
  private inputField: TextInput
  private messageView: Viewport
  private statusBar: StatusBar
  
  view(): string {
    return render(
      this.messageView.view(),
      this.statusBar.view(),
      this.inputField.view()
    )
  }
}

// Reusable components (from Bubbles library)
- TextInput
- Viewport
- Table
- Spinner
- ProgressBar
- Paginator
```

### Performance Optimizations

1. **Frame Rate Control**: Limited redraw rate (60 FPS default)
2. **Viewport**: Only render visible content
3. **Memoization**: Cache rendered strings
4. **Event Debouncing**: Throttle rapid updates
5. **Lazy Loading**: Load sessions on-demand

---

## Monorepo Structure

### Package Organization

OpenCode uses Bun workspaces for monorepo management:

```
opencode/
├── packages/
│   ├── sdk/                    # @opencode-ai/sdk
│   │   ├── src/
│   │   │   ├── index.ts       # Public API
│   │   │   ├── client.ts      # HTTP client
│   │   │   ├── types.ts       # TypeScript types
│   │   │   └── ...
│   │   ├── package.json       # "name": "@opencode-ai/sdk"
│   │   └── tsconfig.json
│   │
│   ├── plugin/                 # @opencode-ai/plugin
│   │   ├── src/
│   │   │   ├── index.ts       # Plugin interface
│   │   │   ├── types.ts       # Hook types
│   │   │   └── ...
│   │   ├── package.json       # "name": "@opencode-ai/plugin"
│   │   └── tsconfig.json
│   │
│   ├── ui/                     # @opencode-ai/ui
│   │   ├── src/
│   │   │   ├── components/
│   │   │   │   ├── Chat.tsx
│   │   │   │   ├── FileExplorer.tsx
│   │   │   │   └── ...
│   │   │   ├── styles/
│   │   │   └── index.ts
│   │   ├── package.json       # "name": "@opencode-ai/ui"
│   │   └── tsconfig.json
│   │
│   └── opencode/               # opencode CLI (default export)
│       ├── src/
│       │   ├── index.ts        # CLI entry point
│       │   ├── server.ts       # HTTP server setup
│       │   ├── cmd/
│       │   │   ├── root.ts
│       │   │   ├── serve.ts
│       │   │   └── ...
│       │   ├── core/
│       │   │   ├── agent.ts
│       │   │   ├── session.ts
│       │   │   ├── tool.ts
│       │   │   └── ...
│       │   ├── route/
│       │   │   ├── session.ts
│       │   │   ├── message.ts
│       │   │   ├── file.ts
│       │   │   └── ...
│       │   ├── plugin/
│       │   │   ├── loader.ts
│       │   │   └── executor.ts
│       │   └── integration/
│       │       ├── mcp.ts
│       │       ├── lsp.ts
│       │       └── providers.ts
│       ├── package.json        # "name": "opencode"
│       └── tsconfig.json
│
├── bun.workspaces.yaml         # Workspace config
├── turbo.json                  # Turbo task orchestration
├── package.json               # Root package
└── README.md
```

### Workspace Dependencies

```json
{
  "dependencies": {
    "@opencode-ai/sdk": "workspace:*",
    "@opencode-ai/plugin": "workspace:*"
  }
}
```

Bun resolves `workspace:*` to local paths during development:
```bash
# Actual resolution
@opencode-ai/sdk -> file:../sdk
@opencode-ai/plugin -> file:../plugin
```

During publishing, versions are replaced:
```bash
# Published version
@opencode-ai/sdk@0.15.18 -> npm
@opencode-ai/plugin@0.15.18 -> npm
```

### Package Purposes & Dependencies

#### SDK Package (`@opencode-ai/sdk`)
**Purpose**: Type-safe HTTP client for OpenCode API
**Dependencies**: None (zero runtime deps!)
**Exports**: 
- `OpencodeClient`
- TypeScript types for all API resources
- Helper functions

**Design Principle**: Minimal bundle size for consuming apps

#### Plugin Package (`@opencode-ai/plugin`)
**Purpose**: Plugin interface and hook definitions
**Dependencies**: 
- `zod` (for schema validation)
- `@opencode-ai/sdk` (for types)

**Exports**:
- `Plugin` interface
- Hook types
- Context types

#### UI Package (`@opencode-ai/ui`)
**Purpose**: Reusable React/SolidJS components
**Dependencies**:
- Framework (React, SolidJS)
- UI library (Kobalte)
- Styling (Tailwind)
- Syntax highlighting (Shiki)

**Exports**:
- `ChatComponent`
- `FileExplorer`
- `AgentSelector`
- etc.

#### OpenCode Package (CLI)
**Purpose**: Main application & HTTP server
**Dependencies**: All others + AI SDK + CLI tools
**Exports**: CLI command, server

### Build Pipeline

```yaml
# turbo.json - Task orchestration
{
  "tasks": {
    "build": {
      "depends": ["^build"],
      "outputs": ["dist/**"],
      "cache": true
    },
    "typecheck": {
      "outputs": [],
      "cache": true
    },
    "test": {
      "depends": ["^build"],
      "outputs": [],
      "cache": false
    }
  }
}
```

Build order (determined by dependencies):
```
1. typecheck (no dependencies, all packages in parallel)
2. build (depends on typecheck)
   - sdk builds first (no deps)
   - plugin builds (depends on sdk)
   - ui builds (depends on sdk)
   - opencode builds (depends on all)
```

### Development Workflow

```bash
# Install dependencies
bun install

# Development with hot reload
bun run dev

# Type checking
bun run typecheck

# Building
bun run build

# Testing
bun test

# Clean
bun run clean
```

### Publishing Pipeline

The system supports 7 distribution channels:

1. **npm** - JavaScript package registry
2. **GitHub Releases** - Binary + source
3. **Docker** - Container image
4. **AUR** - Arch User Repository
5. **Homebrew** - macOS/Linux package manager
6. **VS Code Marketplace** - VSCode extension
7. **Open VSX** - Open VSCode extension registry

All packages use unified versioning (0.15.18 across all):

```typescript
// Version update in CI
const version = '0.15.18'

// Update all package.json files
for (const pkg of ['sdk', 'plugin', 'ui', 'opencode']) {
  updateVersion(`packages/${pkg}/package.json`, version)
}

// Publish to all channels
publish('npm', version)
publish('github', version)
publish('docker', version)
publish('homebrew', version)
// etc.
```

---

## Agent Architecture & Loop

### Agent Types

OpenCode supports two agent categories:

#### Primary Agents
- **build** (default): Full access agent for development
- **plan**: Read-only agent for analysis

#### Subagents
- **general**: Multi-step task execution
- **explore**: Codebase navigation and search

### Agent Loop

The agent loop is the core of OpenCode's operation:

```
┌─────────────────────────────────────────┐
│  1. User Input                          │
│     (Prompt, command, or context)       │
└─────────────┬───────────────────────────┘
              │
              ↓
┌─────────────────────────────────────────┐
│  2. System Prompt Assembly              │
│     - Base instructions                 │
│     - Available tools                   │
│     - Previous context                  │
│     - Current session state             │
└─────────────┬───────────────────────────┘
              │
              ↓
┌─────────────────────────────────────────┐
│  3. LLM Generation                      │
│     - Send to AI provider               │
│     - Stream response                   │
│     - Parse tool calls                  │
└─────────────┬───────────────────────────┘
              │
              ├──→ Text Output?
              │    └─→ Add to message
              │
              ├──→ Tool Call?
              │    └─→ Execute tool
              │
              └──→ Done?
                   └─→ End loop
```

### Detailed Agent Loop Implementation

```typescript
async function runAgentLoop(
  session: Session,
  userInput: string,
  options: {
    maxSteps?: number
    timeout?: number
  } = {}
) {
  // 1. Validate input
  if (!userInput.trim()) {
    throw new Error('Empty input')
  }
  
  // 2. Get model
  const model = await getModel(session.agent.model)
  
  // 3. Build system prompt
  const systemPrompt = buildSystemPrompt({
    agent: session.agent,
    tools: await getSessionTools(session),
    context: session.messages.slice(-5) // Recent context
  })
  
  // 4. Prepare messages
  const messages = [
    ...session.messages,
    { role: 'user', content: userInput }
  ]
  
  // 5. Run agentic loop
  let stepCount = 0
  const maxSteps = options.maxSteps ?? 10
  let done = false
  let textContent = ''
  
  while (!done && stepCount < maxSteps) {
    // A. Stream LLM response
    const { textStream, toolCalls } = await streamText({
      model,
      system: systemPrompt,
      messages,
      tools: toolMap,
      stopWhen: () => stepCount >= maxSteps
    })
    
    // B. Collect text
    for await (const chunk of textStream) {
      textContent += chunk
      
      // Emit in real-time
      eventBus.emit('message.chunk', { chunk })
    }
    
    // C. Execute tools
    for (const toolCall of toolCalls) {
      eventBus.emit('tool.before', { tool: toolCall.toolName })
      
      try {
        // Execute with permission check
        const result = await executeTool(
          toolCall.toolName,
          toolCall.args
        )
        
        // Add to message history
        messages.push({
          role: 'assistant',
          content: [
            { type: 'text', text: textContent },
            { type: 'tool_use', id: toolCall.id, name: toolCall.toolName }
          ]
        })
        
        messages.push({
          role: 'user',
          content: [{
            type: 'tool_result',
            toolUseId: toolCall.id,
            content: result
          }]
        })
        
        textContent = ''
        
        eventBus.emit('tool.after', {
          tool: toolCall.toolName,
          success: true,
          result
        })
      } catch (err) {
        // Tool failed
        messages.push({
          role: 'user',
          content: [{
            type: 'tool_result',
            toolUseId: toolCall.id,
            isError: true,
            content: String(err)
          }]
        })
        
        eventBus.emit('tool.after', {
          tool: toolCall.toolName,
          success: false,
          error: String(err)
        })
      }
    }
    
    // D. Check if done
    if (toolCalls.length === 0) {
      done = true
    }
    
    stepCount++
  }
  
  // 6. Save final message
  await session.messages.push({
    id: uuid(),
    role: 'assistant',
    content: textContent,
    timestamp: new Date()
  })
  
  // 7. Persist session
  await persistSession(session)
  
  // 8. Check for auto-compaction
  await maybeCompactSession(session)
}
```

### System Prompt Construction

```typescript
function buildSystemPrompt(config: {
  agent: Agent
  tools: Tool[]
  context: Message[]
}): string {
  const prompt = [
    // Base instructions
    config.agent.instructions || 'You are a helpful AI coding assistant.',
    
    // Tools available
    `You have access to the following tools:`,
    config.tools.map(t => `- ${t.name}: ${t.description}`).join('\n'),
    
    // Previous context (if any)
    config.context.length > 0 && `Recent context:`,
    config.context.map(m => `${m.role}: ${m.content}`).join('\n'),
    
    // Behavior guidelines
    'Use tools to understand and modify the codebase.',
    'Always verify changes before committing.',
    'Provide clear explanations for your actions.'
  ].filter(Boolean).join('\n\n')
  
  return prompt
}
```

### Subagent Invocation

Primary agents can invoke subagents via the `task` tool:

```typescript
const taskTool = {
  name: 'task',
  description: 'Delegate task to subagent',
  
  schema: {
    type: 'object',
    properties: {
      agent: {
        type: 'string',
        enum: ['general', 'explore'],
        description: 'Which subagent to use'
      },
      prompt: {
        type: 'string',
        description: 'Task description'
      }
    },
    required: ['agent', 'prompt']
  },
  
  execute: async ({ agent, prompt }) => {
    // Create subagent session
    const subSession = {
      ...session,
      id: uuid(),
      agent: getAgent(agent),
      parent: session.id
    }
    
    // Run subagent loop
    const result = await runAgentLoop(subSession, prompt, {
      maxSteps: 5
    })
    
    return result
  }
}
```

---

## Design Patterns

### 1. Event-Driven Architecture

OpenCode uses a **global event bus** for decoupled communication:

```typescript
// Core event emitter
const eventBus = new EventEmitter()

// Subscribe to events
eventBus.on('tool.executed', (event) => {
  // Handle tool execution
  logger.log(`Tool executed: ${event.toolName}`)
  session.addToolResult(event)
  updateUI()
})

// Emit events
eventBus.emit('tool.executed', {
  toolName: 'bash',
  args: { command: 'ls' },
  result: 'file1.js\nfile2.js'
})

// SSE broadcast
eventBus.on('*', (event) => {
  broadcastSSE(event)  // Send to connected clients
})
```

Benefits:
- **Decoupling**: Components don't know about each other
- **Extensibility**: Plugins can hook into any event
- **Observability**: All state changes logged
- **Real-time**: Easy to broadcast updates

### 2. Provider Abstraction

OpenCode abstracts provider complexity through layered wrappers:

```typescript
// Level 3: Application code
async function solveTask(task: string) {
  const model = await getModel('default')
  const response = await streamText({
    model,
    prompt: task,
    tools: availableTools
  })
}

// Level 2: OpenCode wrapper
async function getModel(modelId: string) {
  const [provider, name] = parseModel(modelId)
  const sdk = await loadProviderSDK(provider)
  return sdk.getModel(name)
}

// Level 1: Provider SDK (@ai-sdk/anthropic, etc)
const model = anthropic.getModel('claude-3-5-sonnet')

// Level 0: Provider API
POST https://api.anthropic.com/v1/messages
Authorization: Bearer $ANTHROPIC_API_KEY
```

Benefits:
- **Swappable**: Change provider without code changes
- **Cost-Aware**: Track costs per provider
- **Dynamic Loading**: Install SDKs on-demand
- **Caching**: Share SDK instances

### 3. Storage with Exclusive Locks

File-based storage with atomic operations:

```typescript
// Prevent concurrent writes
async function writeSession(session: Session) {
  const path = `${dir}/${session.id}.json`
  
  // Acquire exclusive lock
  const lock = await acquireLock(path)
  
  try {
    // Write to temporary file
    const tmpPath = path + '.tmp'
    await fs.writeFile(tmpPath, JSON.stringify(session))
    
    // Atomic rename
    await fs.rename(tmpPath, path)
  } finally {
    // Always release lock
    await lock.release()
  }
}
```

Benefits:
- **ACID**: Atomic, Consistent, Isolated, Durable
- **Crash-safe**: Incomplete writes rolled back
- **Scalable**: File system handles concurrency
- **Observable**: Can inspect files directly

### 4. Plugin Hook System

Plugins extend functionality through well-defined hooks:

```typescript
// Plugin system executes hooks in order
async function executeHook(
  hookName: string,
  context: PluginContext,
  ...args: any[]
) {
  for (const plugin of plugins) {
    if (!plugin[hookName]) continue
    
    try {
      // Execute plugin hook
      await plugin[hookName](context, ...args)
    } catch (err) {
      // Log but don't crash
      logger.error(`Plugin ${plugin.name} hook error:`, err)
    }
  }
}
```

Benefits:
- **Extensible**: Add features via plugins
- **Isolated**: Plugin failures don't crash app
- **Composable**: Multiple plugins work together
- **Observable**: All hooks logged

### 5. Graceful Degradation

OpenCode handles resource constraints gracefully:

```typescript
// Auto-compaction when context approaches limit
if (session.tokenUsage > contextLimit * 0.95) {
  await compactSession(session)
}

// Truncate output if too long
let output = toolResult
if (output.length > MAX_OUTPUT) {
  output = output.slice(0, MAX_OUTPUT) + '\n[...truncated...]'
}

// Fallback models if preferred unavailable
async function getModel(preferred: string) {
  try {
    return await loadProvider(preferred)
  } catch (err) {
    logger.warn(`Preferred model unavailable, using fallback`)
    return await loadProvider('fallback-model')
  }
}
```

Benefits:
- **Robustness**: Handles edge cases
- **User-Aware**: Notifies of limitations
- **Cost-Aware**: Manages token usage
- **Resilient**: Continues despite failures

### 6. Middleware Pattern

Composable request/response processing:

```typescript
// Chain middleware
const app = new Hono()

// Logging middleware
app.use('*', async (c, next) => {
  console.time(c.req.path)
  await next()
  console.timeEnd(c.req.path)
})

// Auth middleware
app.use('/api/*', async (c, next) => {
  const token = c.req.header('Authorization')
  if (!token) return c.text('Unauthorized', 401)
  c.set('user', verifyToken(token))
  await next()
})

// CORS middleware
app.use('*', cors())
```

Benefits:
- **Composable**: Stack middleware in order
- **Reusable**: Middleware shared across routes
- **Testable**: Mock middleware for testing
- **Clean**: Separation of concerns

---

## Best Practices for Building

### 1. Architecture & Design

**Single Responsibility**
```typescript
// ❌ Bad: Multiple concerns
async function handleSession(req: Request) {
  const session = parseRequest(req)
  const result = runAgent(session)
  await saveDatabase(result)
  await sendEmail(result)
  return new Response(result)
}

// ✅ Good: Single concern
async function handleSession(req: Request) {
  const session = parseRequest(req)
  const result = await runAgent(session)
  await persistSession(result)
  return new Response(result)
}

// Email sending is separate concern
eventBus.on('session.completed', async (session) => {
  await sendNotification(session)
})
```

**Dependency Injection**
```typescript
// ❌ Bad: Hard to test, tightly coupled
async function runAgent(prompt: string) {
  const model = await getModel('claude-3-5-sonnet')
  const tools = await loadTools()
  const storage = new FileStorage()
  // ...
}

// ✅ Good: Testable, decoupled
async function runAgent(
  prompt: string,
  options: {
    model: Model
    tools: Tool[]
    storage: Storage
  }
) {
  // Use injected dependencies
  const result = await options.model.generate(prompt)
  await options.storage.save(result)
}
```

### 2. Plugin Development

**Clear Contract**
```typescript
// Define plugin interface clearly
interface CustomPlugin {
  name: string
  version: string
  capabilities: string[]
  
  // Required methods
  initialize(ctx: PluginContext): Promise<void>
  shutdown(): Promise<void>
  
  // Optional hooks
  onToolExecuted?(result: ToolResult): Promise<void>
}
```

**Error Handling**
```typescript
// ❌ Bad: Plugin crashes app
export default {
  async onToolExecuted(ctx, result) {
    const response = await fetch(result.data)
    const data = await response.json()
    ctx.client.log(JSON.stringify(data))  // No error handling!
  }
}

// ✅ Good: Graceful error handling
export default {
  async onToolExecuted(ctx, result) {
    try {
      const response = await fetch(result.data)
      if (!response.ok) {
        ctx.client.warn(`Failed to fetch: ${response.status}`)
        return
      }
      
      const data = await response.json()
      ctx.client.log(JSON.stringify(data))
    } catch (err) {
      ctx.client.error(`Plugin error: ${err.message}`)
    }
  }
}
```

### 3. Tool Development

**Schema Validation**
```typescript
import { z } from 'zod'

// Define schema with Zod
const schema = z.object({
  filePath: z.string().describe('Path to file'),
  pattern: z.string().describe('Search pattern'),
  caseSensitive: z.boolean().default(false).describe('Case sensitive search')
})

// Tool with validated inputs
const searchTool = {
  name: 'search-file',
  description: 'Search file content',
  
  schema: {
    type: 'object',
    properties: {
      filePath: { type: 'string' },
      pattern: { type: 'string' },
      caseSensitive: { type: 'boolean', default: false }
    },
    required: ['filePath', 'pattern']
  },
  
  execute: async (input) => {
    // Validate
    const validated = schema.parse(input)
    
    // Execute
    const results = await searchFile(
      validated.filePath,
      validated.pattern,
      validated.caseSensitive
    )
    
    return JSON.stringify(results)
  }
}
```

**Output Format**
```typescript
// ❌ Bad: Inconsistent output
execute: async ({ file }) => {
  const content = await read(file)
  return content  // Could be HTML, JSON, binary
}

// ✅ Good: Consistent format
execute: async ({ file }) => {
  const content = await read(file)
  return JSON.stringify({
    file,
    size: content.length,
    preview: content.slice(0, 500),
    success: true
  })
}
```

### 4. Session Management

**Context Window Management**
```typescript
// Monitor token usage
async function monitorSession(session: Session) {
  const usage = calculateTokenUsage(session)
  const limit = session.model.contextWindow
  const percentage = (usage / limit) * 100
  
  if (percentage > 90) {
    logger.warn(`Session ${session.id} at ${percentage}% context`)
  }
  
  if (percentage > 95) {
    logger.info(`Auto-compacting session ${session.id}`)
    await compactSession(session)
  }
}
```

**Cost Tracking**
```typescript
// Track costs per session
async function trackCost(
  session: Session,
  usage: { inputTokens: number; outputTokens: number }
) {
  const pricing = getPricing(session.model)
  const cost = (usage.inputTokens * pricing.input) +
               (usage.outputTokens * pricing.output)
  
  session.totalCost += cost
  
  // Warn if exceeds budget
  if (session.totalCost > 50) {
    logger.warn(`Session ${session.id} cost exceeds $50`)
  }
}
```

### 5. Error Handling

**Graceful Degradation**
```typescript
// ❌ Bad: Crashes on provider failure
async function getModel(provider: string) {
  return await loadProvider(provider)
}

// ✅ Good: Falls back on failure
async function getModel(preferred: string) {
  try {
    return await loadProvider(preferred)
  } catch (err) {
    logger.warn(`Failed to load ${preferred}, using fallback`)
    return await loadProvider('gpt-4o-mini')
  }
}
```

**User Communication**
```typescript
// ❌ Bad: Silent failure
async function executeCommand(cmd: string) {
  try {
    return await exec(cmd)
  } catch (err) {
    logger.error(err)
  }
}

// ✅ Good: Inform user
async function executeCommand(cmd: string, ctx) {
  try {
    const result = await exec(cmd)
    ctx.client.success(`Command executed`)
    return result
  } catch (err) {
    ctx.client.error(`Command failed: ${err.message}`)
    throw err
  }
}
```

### 6. Testing

**Plugin Testing**
```typescript
import { test, expect } from 'bun:test'

test('plugin executes without errors', async () => {
  const mockContext = {
    project: { name: 'test', path: '/test' },
    client: { log: () => {}, warn: () => {}, error: () => {} },
    directory: '/test'
  }
  
  const plugin = (await import('./plugin.ts')).default
  
  await plugin.onToolExecuted?.(mockContext, {
    toolName: 'test',
    result: 'success'
  })
  
  expect(true).toBe(true)
})
```

---

## Integration Patterns

### Pattern 1: CLI Integration

```typescript
// Integrate OpenCode as CLI tool
import { OpencodeClient } from '@opencode-ai/sdk'

async function main() {
  const client = new OpencodeClient({
    baseURL: process.env.OPENCODE_URL || 'http://localhost:4096'
  })
  
  // Create session
  const session = await client.session.new({
    title: 'CLI Task'
  })
  
  // Send task
  const response = await client.session.prompt(session.id, {
    text: process.argv[2] || 'Help me'
  })
  
  console.log(response.text)
}

main()
```

### Pattern 2: IDE Plugin

OpenCode can be integrated into IDEs via the SDK:

```typescript
// VS Code extension example
import * as vscode from 'vscode'
import { OpencodeClient } from '@opencode-ai/sdk'

export function activate(context: vscode.ExtensionContext) {
  const client = new OpencodeClient()
  
  // Register command
  const disposable = vscode.commands.registerCommand(
    'opencode.review',
    async () => {
      const editor = vscode.window.activeTextEditor
      if (!editor) return
      
      const session = await client.session.new({
        title: 'Code Review'
      })
      
      const response = await client.session.prompt(session.id, {
        text: `Review this code:\n\n${editor.document.getText()}`
      })
      
      vscode.window.showInformationMessage(response.text)
    }
  )
  
  context.subscriptions.push(disposable)
}
```

### Pattern 3: Web Dashboard

```typescript
// React component for OpenCode dashboard
import React, { useState, useEffect } from 'react'
import { OpencodeClient } from '@opencode-ai/sdk'

export function OpencodePanel() {
  const [sessions, setSessions] = useState([])
  const [client] = useState(() => new OpencodeClient())
  
  useEffect(() => {
    // Load sessions on mount
    client.session.list().then(setSessions)
  }, [client])
  
  return (
    <div>
      <h1>OpenCode Sessions</h1>
      {sessions.map(session => (
        <div key={session.id}>
          <h3>{session.title}</h3>
          <p>Messages: {session.messages.length}</p>
        </div>
      ))}
    </div>
  )
}
```

---

## Deployment & Infrastructure

### Containerization

```dockerfile
# Deploy OpenCode in Docker
FROM oven/bun:latest

WORKDIR /app

# Copy source
COPY . .

# Install dependencies
RUN bun install

# Expose API
EXPOSE 4096

# Start server
CMD ["bun", "run", "src/index.ts"]
```

### Environment Configuration

```bash
# .env.example
OPENCODE_PORT=4096
OPENCODE_HOSTNAME=0.0.0.0

# LLM Provider
OPENCODE_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...

# Optional: Remote server
OPENCODE_DATA_DIR=/data/opencode

# Logging
DEBUG=opencode:*
LOG_LEVEL=info
```

### Multi-Provider Setup

```json
{
  "provider": {
    "anthropic": {
      "apiKey": "${ANTHROPIC_API_KEY}"
    },
    "openai": {
      "apiKey": "${OPENAI_API_KEY}"
    },
    "groq": {
      "apiKey": "${GROQ_API_KEY}"
    }
  },
  "models": {
    "default": "claude-3-5-sonnet-20241022",
    "fast": "gpt-4o-mini",
    "local": "ollama:mistral"
  }
}
```

### Scaling Considerations

1. **Session Storage**: Move to database for scale
2. **Event Bus**: Use Redis for distributed events
3. **Tool Execution**: Queue-based execution
4. **Multi-Server**: Load balance HTTP requests
5. **Rate Limiting**: Implement per-user limits

---

## Conclusion

OpenCode's architecture represents a sophisticated, production-grade AI agent framework with:

1. **Separation of Concerns**: HTTP server decoupled from UI
2. **Extensibility**: Plugin system, MCP servers, custom tools
3. **Provider Agnosticism**: Support for 75+ LLM providers
4. **Event-Driven**: Real-time updates via SSE
5. **Resource Management**: Context tracking, auto-compaction
6. **Scalability**: Monorepo structure, multiple distribution channels

### Key Architectural Advantages

✅ **Flexibility**: Run server and client on different machines  
✅ **Testability**: Test HTTP endpoints independently  
✅ **Extensibility**: Plugins extend without forking  
✅ **Integration**: Multiple ways to interface (CLI, web, IDE)  
✅ **Production-Ready**: Atomic storage, error handling, cost tracking  

### When Building on OpenCode

1. **For New Tools**: Develop as plugins or MCP servers
2. **For Integration**: Use the OpenCode SDK with type safety
3. **For Extension**: Hook into plugin events
4. **For Customization**: Configure agents and permissions
5. **For Scale**: Understand storage and event patterns

OpenCode proves that an open-source, well-architected AI development platform can match or exceed proprietary solutions in functionality while maintaining extensibility and user control.

---

**Document Version**: 1.0  
**Last Updated**: December 2025  
**Maintained By**: OpenCode Community  
**License**: MIT (same as OpenCode)
