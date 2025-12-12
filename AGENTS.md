# Open Swarm - Multi-Agent Coordination Framework

## ⚠️ CRITICAL: READ THIS FIRST ⚠️

**Before doing ANY work on this project, you MUST understand these non-negotiable rules:**

### 🔴 RULE #1: BEADS IS MANDATORY FOR ALL CHANGES
- **EVERY code change** requires a Beads task
- **EVERY bug fix** requires a Beads task  
- **EVERY feature** requires a Beads task
- **EVERY refactor** requires a Beads task
- **NO EXCEPTIONS**

If there is no Beads task ID (e.g., `open-swarm-xyz`), **DO NOT** make any changes.

### 🔴 RULE #2: SERENA IS THE ONLY WAY TO EDIT CODE
- **ALL Go code editing** must use Serena's semantic tools
- **NEVER** use Read + Edit for `.go` files
- **NEVER** use bash tools (`sed`, `awk`) for code
- **USE:** `serena_find_symbol`, `serena_replace_symbol_body`, `serena_insert_after_symbol`, `serena_rename_symbol`

**Exception:** Non-code files (`.md`, `.json`, `.yaml`) can use Edit tool.

### ✅ Correct Workflow (MANDATORY)
1. **Get/Create Beads task** → `bd create` or `bd ready --json`
2. **Start task** → `bd update task-id --status in_progress`
3. **Navigate with Serena** → `serena_find_symbol`, `serena_find_referencing_symbols`
4. **Edit with Serena** → `serena_replace_symbol_body` or `serena_insert_after_symbol`
5. **Complete task** → `bd close task-id --reason "description"`

**Violating these rules will result in broken coordination and merge conflicts.**

---

## Project Overview

Open Swarm is a Go-based multi-agent coordination framework leveraging:
- **Agent Mail MCP Server** - Git-backed messaging and file reservations
- **Beads** - Lightweight Git-backed issue tracking (CRITICAL for coordination)
- **Serena MCP Server** - LSP-powered semantic code navigation (MANDATORY for editing)

This project enables multiple AI agents to work collaboratively on the same codebase without conflicts.

## Architecture

```
open-swarm/
├── cmd/open-swarm/        # CLI tool entry point
├── pkg/
│   ├── coordinator/       # Multi-agent coordination logic
│   ├── agent/            # Agent identity and management
│   └── tasks/            # Task management integration
├── internal/
│   └── config/           # Configuration handling
├── .beads/               # Beads issue tracking (gitignored SQLite, committed JSONL)
├── .opencode/            # OpenCode extensions
│   ├── tool/            # Custom MCP tools
│   ├── plugin/          # OpenCode plugins
│   ├── agent/           # Custom agent definitions
│   └── command/         # Slash command definitions
├── opencode.json         # OpenCode configuration
└── AGENTS.md            # This file
```

## Prerequisites

- **Go 1.25+** - `go version`
- **Agent Mail** - `python -m mcp_agent_mail.server` (installed at `~/.agent-mail/`)
- **Beads** - `bd --version` (installed via curl script or npm)
- **Serena** - `uvx --from git+https://github.com/oraios/serena serena start-mcp-server --help`
- **OpenCode** - `opencode --version` (SST's opencode, not Claude Code)

## Setup

### Initial Installation

```bash
# Install Go dependencies
go mod download

# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Initialize Beads (if not already done)
bd init

# Start Agent Mail server (in separate terminal)
am  # or: python -m mcp_agent_mail.server

# Verify MCP servers
opencode  # Should auto-connect to Agent Mail and Serena
```

### Building

```bash
# Development build
go build -o bin/open-swarm ./cmd/open-swarm

# Production build with optimizations
go build -ldflags="-s -w" -o bin/open-swarm ./cmd/open-swarm

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Multi-Agent Coordination Workflow

### 🚨 CRITICAL RULES - READ FIRST 🚨

**These rules are MANDATORY for ALL work on this project:**

#### Rule 1: Beads is Non-Negotiable
**EVERY SINGLE CODE CHANGE, NO MATTER HOW SMALL, MUST BE TRACKED IN BEADS.**

- Creating a function? **Beads task first.**
- Fixing a typo? **Beads task first.**
- Refactoring code? **Beads task first.**
- Adding tests? **Beads task first.**
- Writing documentation? **Beads task first.**

**If there is no Beads task, DO NOT make the change.**

#### Rule 2: Serena is the ONLY Way to Edit Code
**ALL code file editing MUST go through Serena's semantic tools.**

- ✅ **USE:** `serena_find_symbol`, `serena_replace_symbol_body`, `serena_insert_after_symbol`
- ❌ **NEVER USE:** Direct file editing, Read + Edit tools, bash `sed`/`awk`

**Exception:** Non-code files only (markdown, JSON, YAML, config files). Use Edit tool for those.

#### Rule 3: Understand Before You Change
**ALWAYS use Serena to understand code before modifying it:**

1. **Find the symbol:** `serena_find_symbol: "FunctionName"`
2. **Check usage:** `serena_find_referencing_symbols: "FunctionName"`
3. **Get context:** `serena_get_symbols_overview: "path/to/file.go"`
4. **Only then edit:** `serena_replace_symbol_body` or `serena_insert_after_symbol`

### Workflow Summary

**MANDATORY workflow for ALL code work:**

1. **Task in Beads** - Create or claim a Beads task for the work
2. **Serena for Navigation** - Use Serena to understand the code structure
3. **Serena for Editing** - Use Serena's symbol-level tools to modify code
4. **Never edit blindly** - Always understand impact before making changes

### Session Start Protocol

**Every agent should start their session with:**

```bash
# 1. Check for ready work in Beads
bd ready --json

# 2. Start OpenCode
opencode

# 3. In OpenCode, sync with Agent Mail
/sync

# 4. Check inbox for messages
Check Agent Mail inbox for coordination messages

# 5. Select task and reserve files
/task-start bd-xxxx
```

### Agent Registration

Each agent registers with Agent Mail using absolute project path as the identity key:

```
Project Key: /home/lewis/src/open-swarm
Program: opencode
Model: anthropic/claude-sonnet-4-5
Task Description: [Current work focus]
```

Agent names are auto-generated (e.g., "BlueLake", "GreenForest", "RedMountain").

### File Reservation Strategy

**Before editing ANY files, reserve them:**

```
Reserve files: <pattern>
Exclusive: true
TTL: 3600 seconds (1 hour)
Reason: [Brief description]
```

**Common reservation patterns:**
- Backend API work: `internal/api/**/*.go`, `pkg/handlers/**/*.go`
- Database work: `internal/infrastructure/**/*.go`, `pkg/db/**/*.go`
- Configuration: `internal/config/**/*.go`, `opencode.json`
- Tests: `**/*_test.go`, `test/**/*.go`
- Documentation: `docs/**/*.md`, `README.md`

**Always release when done:**
```
/release
```

### Task Management with Beads

**MANDATORY: All code changes must be tracked in Beads.**

Never make code edits without a corresponding Beads task. This ensures:
- Work is coordinated across agents
- Changes are traceable
- Dependencies are managed
- Progress is visible to the team

**Check ready work:**
```bash
bd ready --json
```

**Start task:**
```bash
bd update bd-xxxx --status in_progress
```

**During development:**
```bash
# File discovered issues (ALWAYS create sub-tasks for new work found)
bd create "Issue description" --parent bd-xxxx --type discovered-from

# Add dependencies when work blocks other work
bd dep add bd-child bd-parent --type blocks
```

**Complete task:**
```bash
bd close bd-xxxx --reason "Implemented and tested"
```

**Task-Thread Linking:**
Use Beads issue IDs as Agent Mail thread IDs (e.g., `thread: bd-a1b2`) to maintain traceability.

### Messaging Patterns

**Notify completion of blocking work:**
```
Send message to: <AgentName>
Subject: [bd-xxxx] Backend endpoints ready
Body: Implemented CRUD endpoints at /api/v1/users. Ready for frontend integration.
Thread: bd-xxxx
Importance: normal
Ack Required: true
```

**Request help:**
```
Send message to: <AgentName>
Subject: Help needed with database migration
Body: Encountering issues with foreign key constraints in migration 003. Can you review?
Thread: db-migration-help
Importance: high
```

**Broadcast status:**
```
Send message to: All agents
Subject: [bd-yyyy] Authentication system complete
Body: JWT authentication implemented and tested. All agents can now use /api/auth endpoints.
Thread: bd-yyyy
```

## Code Standards

### Go Best Practices

**Idiomatic Go:**
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (automatic via formatter config)
- Run `golangci-lint run` before committing
- Use short, descriptive variable names
- Write self-documenting code with clear function names

**Architecture:**
- Apply Clean Architecture with clear layer separation
- Handler → Service → Repository pattern
- Interface-driven development (depend on interfaces, not concrete types)
- Explicit dependency injection via constructors
- No global state (use constructor functions)

**Error Handling:**
```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// Good: Custom domain errors
var ErrUserNotFound = errors.New("user not found")

// Good: Type assertion for specific handling
if errors.Is(err, sql.ErrNoRows) {
    return ErrUserNotFound
}
```

**Testing:**
```go
// Table-driven tests
func TestUserService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *User
        want    error
        wantErr bool
    }{
        {"valid user", &User{Name: "John"}, nil, false},
        {"empty name", &User{Name: ""}, ErrInvalidInput, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

**Dependency Injection:**
```go
// Constructor pattern with interfaces
func NewUserService(repo UserRepository, logger *log.Logger) *UserService {
    return &UserService{
        repo:   repo,  // Interface, not concrete type
        logger: logger,
    }
}
```

### Code Quality Requirements

- **Test Coverage:** Minimum 80% on critical paths
- **Linting:** Must pass `golangci-lint run` with zero errors
- **Formatting:** All code must be `gofmt`-formatted
- **Documentation:** GoDoc comments on all exported symbols
- **Error Handling:** Never ignore errors
- **Complexity:** Keep cyclomatic complexity low (max 15 per function)

## Development Workflows

### Adding a New Feature

**Multi-Agent Parallel Workflow (with Beads + Serena):**

**Agent 1 (Backend):**
```bash
# 1. Check Beads for ready tasks
bd ready --json  # Select bd-a1b2 (API implementation)

# 2. Reserve files via Agent Mail
Reserve: internal/api/**/*.go, pkg/domain/**/*.go
Exclusive: true

# 3. Update Beads status
bd update bd-a1b2 --status in_progress

# 4. Use Serena to understand existing code structure
Find symbol: "APIHandler"  # Locate existing patterns
Get symbols overview: "internal/api/handlers.go" depth 1
Find references: "UserService"  # Understand usage

# 5. Implement feature using Serena's editing tools
# NEVER directly edit files - use Serena!

# Example: Add new handler method
Insert after symbol: "APIHandler.GetUser" with body:
```go
func (h *APIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

# Add route registration
Find symbol: "RegisterRoutes"
Replace symbol body: "RegisterRoutes" to add new route

# 6. Test implementation
go test ./internal/api/... -v

# 7. File sub-tasks for any discovered work
bd create "Add input validation for CreateUser" --parent bd-a1b2

# 8. Notify frontend agent via Agent Mail
Send message to: FrontendAgent
Subject: [bd-a1b2] User API endpoints ready
Thread: bd-a1b2

# 9. Complete in Beads and release reservations
bd close bd-a1b2 --reason "Implemented and tested using Serena"
/release
```

**Agent 2 (Frontend/Integration):**
```bash
# 1. Wait for backend notification
Check inbox  # See message from Agent 1

# 2. Reserve frontend files
Reserve: web/src/services/**/*.ts

# 3. Implement integration
# ... development work ...

# 4. Reply
Reply to message: [bd-a1b2] Frontend integration complete
Thread: bd-a1b2

# 5. Release
/release
```

### Bug Fix Workflow (with Beads + Serena)

**MANDATORY: All bug fixes MUST be tracked in Beads and edited via Serena.**

```bash
# 1. File issue in Beads FIRST
bd create "Fix null pointer in user handler" -t bug -p high
# Returns: bd-c3d4

# 2. Start the task
bd update bd-c3d4 --status in_progress

# 3. Reserve affected files via Agent Mail
Reserve: internal/api/user_handlers.go
Exclusive: true

# 4. Use Serena to understand the bug
# NEVER read entire file - use symbol navigation
Find symbol: "HandleUserCreate"
# Examine the returned symbol body

Find references: "User"
# See all usages to understand the problem

# 5. Fix using Serena's editing tools
# NEVER directly edit - use replace_symbol_body
Replace symbol body: "HandleUserCreate" with fixed implementation:
```go
func HandleUserCreate(w http.ResponseWriter, r *http.Request) {
    if r.Body == nil {
        http.Error(w, "request body required", http.StatusBadRequest)
        return
    }
    // ... rest of implementation with null checks
}
```

# 6. Write regression test using Serena
Insert after symbol: "TestHandleUserCreate" with new test:
```go
func TestHandleUserCreate_NullBody(t *testing.T) {
    // Test implementation
}
```

# 7. Run tests
go test ./internal/api/... -v

# 8. Close issue in Beads
bd close bd-c3d4 --reason "Fixed null pointer with null checks, added regression test"

# 9. Release file reservation
/release

# 10. Notify team via Agent Mail if needed
Send message: "Bug bd-c3d4 fixed in HandleUserCreate"
```

**Key Points:**
- Beads task BEFORE any code change
- Serena ONLY for code reading and editing
- Test the fix before closing the task
- Always add regression tests for bugs

### Code Review Workflow

```bash
# Use the reviewer agent
/review internal/api/user_handlers.go

# Or invoke directly
@reviewer Review this file for security issues and Go best practices
```

## Using Custom Commands

OpenCode provides slash commands configured in `opencode.json`:

- `/sync` - Register agent and sync with Agent Mail
- `/reserve <pattern>` - Reserve files for exclusive editing
- `/release` - Release all file reservations
- `/task-ready` - Check Beads for unblocked work
- `/task-start <id>` - Start Beads task and reserve files
- `/task-complete <id>` - Complete task and release files
- `/review <files>` - Code review with @reviewer agent

## Using Serena for Code Navigation and Editing

**MANDATORY: Use Serena for ALL code modifications.**

Serena provides LSP-powered semantic navigation and editing. You MUST use these tools instead of direct file editing.

### Why Use Serena?

1. **Semantic Understanding** - Navigate code by symbols, not lines
2. **Context Efficiency** - Minimize token usage by reading only relevant symbols
3. **Safe Refactoring** - Understand impact before making changes
4. **Precision** - Symbol-level edits prevent accidental breakage
5. **Multi-Agent Coordination** - Other agents can see what you're modifying

### Navigation (Always First Step)

**BEFORE editing, always explore:**

```
# Find the symbol you need to modify
Find symbol: "UserService"
Find symbol: "HandleUserCreate"

# Understand where it's used
Find all references to: "UserService"
Find all references to: "HandleUserCreate"

# Get file overview
Get symbols overview: "internal/api/handlers.go"
```

### Editing (Use Symbol-Level Tools)

**NEVER use direct file editing. ALWAYS use Serena's editing tools:**

```
# Replace a function/method body
Replace symbol body: "HandleUserCreate" with new implementation

# Add a new method to a class/struct
Insert after symbol: "UserService.GetUser" a new method

# Add a new function
Insert before symbol: "FirstFunctionInFile" a new function

# Rename across entire codebase
Rename symbol: "UserService" to "UserManager"
```

### Code Editing Workflow

**MANDATORY STEPS for every code change:**

1. **Understand First** - Use `find_symbol` to locate what you need
2. **Check Impact** - Use `find_references` to see where it's used
3. **Plan Changes** - Determine which symbols need modification
4. **Edit Precisely** - Use `replace_symbol_body` or `insert_after_symbol`
5. **Verify** - Run tests to confirm changes work

### Example: Adding a New Method

```
# 1. Find the struct/class
Find symbol: "UserService"
# Returns: Location in pkg/service/user.go

# 2. Understand existing methods
Get symbols overview: "pkg/service/user.go" with depth 1
# Returns: List of methods on UserService

# 3. Insert new method
Insert after symbol: "UserService.GetUser" with body:
```go
func (us *UserService) UpdateUser(ctx context.Context, id string, updates *UserUpdates) error {
    // Implementation
}
```

# 4. Test
Run: go test ./pkg/service/... -v
```

### When NOT to Use Direct File Editing

**NEVER directly edit files when:**
- Adding/modifying functions, methods, or types
- Refactoring code structure
- Renaming symbols
- Understanding code flow
- Working with large files

**Only use direct file editing for:**
- Non-code files (markdown, JSON, YAML)
- Configuration files
- Very small, isolated changes to non-symbol content

### Benefits of Serena-First Approach:

- **No context waste** - Only read symbols you need, not entire files
- **Precision** - Edit exactly what you intend, nothing more
- **Safety** - LSP ensures semantic correctness
- **Coordination** - Other agents can track symbol-level changes
- **Speed** - Faster than reading/editing large files manually

## Testing Strategy

### Unit Tests
- Test each layer independently (handlers, services, repositories)
- Mock all external dependencies
- Use table-driven patterns
- Run with: `go test ./... -short`

### Integration Tests
- Test interactions between components
- Use real database (Docker for local)
- Test API endpoints end-to-end
- Run with: `go test ./... -tags=integration`

### Test Organization
```
pkg/
  user/
    user.go
    user_test.go          # Unit tests
    user_integration_test.go  # Integration tests
```

## Session End Protocol

**Every agent must complete these steps before ending session:**

1. **Update Beads:**
   ```bash
   bd update <task-id> --status [done|in_progress|blocked]
   # If done:
   bd close <task-id> --reason "Description"
   ```

2. **File new issues for incomplete work:**
   ```bash
   bd create "Remaining work description" --parent <task-id>
   ```

3. **Release file reservations:**
   ```
   /release
   ```

4. **Send status update if needed:**
   ```
   Send message about current state to team
   ```

5. **Sync Git:**
   ```bash
   git add .beads/issues.jsonl
   git commit -m "Update task tracking"
   git push
   ```

## Environment Variables

```bash
# Required
ANTHROPIC_API_KEY=sk-ant-...

# Optional
OPENAI_API_KEY=sk-...
GOOGLE_GENERATIVE_AI_API_KEY=...

# Agent Mail (uses defaults if not set)
AGENT_MAIL_DB=$HOME/.agent-mail/mail.db
AGENT_MAIL_ARCHIVE=$HOME/.agent-mail/archive
```

## Troubleshooting

### Agent Mail Not Connecting

```bash
# Check if server is running
curl http://localhost:8765/health

# Start manually
am  # or: python -m mcp_agent_mail.server

# Check logs
tail -f ~/.agent-mail/server.log
```

### Beads Sync Issues

```bash
# Force sync
bd sync

# Check status
bd list --json

# Re-initialize if needed
bd init
```

### File Reservation Conflicts

```
# Check who has reservation
Query Agent Mail for active reservations

# Options:
# 1. Wait for TTL expiry (1 hour default)
# 2. Contact holding agent via message
# 3. Work on different files
```

## Common Patterns

### Cross-Agent Handoff

```
Agent A completes → Sends message with thread ID
                 ↓
Agent B receives → Joins thread → Reserves files → Implements
                 ↓
Agent B completes → Replies in thread → Releases files
```

### Parallel Development

```
Task decomposition (via Beads)
    ↓
Agent A: Backend (reserves internal/api/**)
Agent B: Frontend (reserves web/src/**)
Agent C: Tests (reserves test/**)
    ↓
All agents work independently
    ↓
Integration phase (coordinate via messages)
```

### Conflict Resolution

```
Reservation conflict detected
    ↓
Check Agent Mail for holder
    ↓
Send message: "Need to edit file X, ETA?"
    ↓
Coordinate handoff or work on different files
```

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Agent Mail Documentation](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [Beads Documentation](https://github.com/steveyegge/beads)
- [Serena Documentation](https://oraios.github.io/serena/)
- [OpenCode Documentation](https://opencode.ai/docs/)

## Quick Reference

| Task | Command | Tool |
|------|---------|------|
| Start session | `opencode` → `/sync` | OpenCode + Agent Mail |
| Check ready work | `bd ready --json` | Beads |
| Start task | `bd update bd-xxxx --status in_progress` | Beads |
| Reserve files | `/reserve pattern` | Agent Mail |
| Check inbox | Check Agent Mail inbox | Agent Mail |
| Send message | Use Agent Mail send_message tool | Agent Mail |
| Find symbol | `find_symbol: "SymbolName"` | **Serena (MANDATORY)** |
| Find references | `find_references: "SymbolName"` | **Serena (MANDATORY)** |
| Get overview | `get_symbols_overview: "file.go"` | **Serena (MANDATORY)** |
| Edit code | `replace_symbol_body` / `insert_after_symbol` | **Serena (MANDATORY)** |
| Rename symbol | `rename_symbol: "OldName" to "NewName"` | **Serena (MANDATORY)** |
| Run tests | `go test ./...` | Go |
| Complete task | `bd close bd-xxxx --reason "..."` | Beads |
| Release files | `/release` | Agent Mail |
| Build | `go build -o bin/open-swarm ./cmd/open-swarm` | Go |

**Critical Reminders:**
- ✅ **ALWAYS** use Beads for task tracking
- ✅ **ALWAYS** use Serena for code navigation and editing
- ✅ **ALWAYS** reserve files before editing
- ❌ **NEVER** edit code files directly without Serena
- ❌ **NEVER** make code changes without a Beads task
