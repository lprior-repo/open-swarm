# Open Swarm - Multi-Agent Coordination Framework

## Project Overview

Open Swarm is a Go-based multi-agent coordination framework leveraging:
- **Agent Mail MCP Server** - Git-backed messaging and file reservations
- **Beads** - Lightweight Git-backed issue tracking
- **Serena MCP Server** - LSP-powered semantic code navigation

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
# File discovered issues
bd create "Issue description" --parent bd-xxxx --type discovered-from

# Add dependencies
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

**Multi-Agent Parallel Workflow:**

**Agent 1 (Backend):**
```bash
# 1. Check Beads
bd ready --json  # Select bd-a1b2 (API implementation)

# 2. Reserve files
Reserve: internal/api/**/*.go, pkg/domain/**/*.go
Exclusive: true

# 3. Update Beads
bd update bd-a1b2 --status in_progress

# 4. Use Serena for navigation
Find symbol: "APIHandler"  # Locate existing patterns
Find references: "UserService"  # Understand usage

# 5. Implement feature
# ... development work ...

# 6. Test
go test ./internal/api/... -v

# 7. Notify frontend agent
Send message to: FrontendAgent
Subject: [bd-a1b2] User API endpoints ready
Thread: bd-a1b2

# 8. Complete
bd close bd-a1b2 --reason "Implemented and tested"
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

### Bug Fix Workflow

```bash
# 1. File issue in Beads
bd create "Fix null pointer in user handler" -t bug -p high

# 2. Reserve affected files
Reserve: internal/api/user_handlers.go

# 3. Use Serena to understand context
Find symbol: "HandleUserCreate"
Find references: "User"

# 4. Fix and test
# ... fix implementation ...
go test ./internal/api/... -v

# 5. Write regression test
# ... test implementation ...

# 6. Close issue
bd close bd-xxxx --reason "Fixed null pointer, added regression test"

# 7. Release
/release
```

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

## Using Serena for Code Navigation

Serena provides LSP-powered semantic navigation:

**Find symbols:**
```
Find symbol: "UserService"
Find symbol: "HandleUserCreate"
```

**Find references:**
```
Find all references to: "User"
Find all references to: "UserRepository"
```

**Symbol-level editing:**
```
Edit symbol: "HandleUserCreate" to add validation
Insert after symbol: "UserService.constructor" a new method
```

**Benefits:**
- Navigate large files without reading entirely
- Understand impact of changes before making them
- Minimize context window usage
- Work at the semantic level, not line-by-line

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

| Task | Command |
|------|---------|
| Start session | `opencode` → `/sync` |
| Check ready work | `/task-ready` or `bd ready --json` |
| Start task | `/task-start bd-xxxx` |
| Reserve files | `/reserve pattern` |
| Check inbox | Check Agent Mail inbox |
| Send message | Use Agent Mail send_message tool |
| Code review | `/review files` |
| Complete task | `/task-complete bd-xxxx` |
| Release files | `/release` |
| Find symbol | Use Serena find_symbol |
| Run tests | `go test ./...` |
| Build | `go build -o bin/open-swarm ./cmd/open-swarm` |
