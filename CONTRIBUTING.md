# Contributing to Open Swarm

Welcome to Open Swarm! This guide explains how to contribute to our multi-agent coordination framework for Go projects. We use a collaborative workflow with OpenCode, Agent Mail, and Beads to enable multiple AI agents to work on the same codebase efficiently.

## Table of Contents

- [Code Style Guide](#code-style-guide)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Beads Workflow for Contributors](#beads-workflow-for-contributors)
- [Development Setup](#development-setup)
- [Adding New Workflows](#adding-new-workflows)
- [Adding New Activities](#adding-new-activities)
- [Multi-Agent Coordination](#multi-agent-coordination)
- [Troubleshooting](#troubleshooting)

## Code Style Guide

### Go Best Practices

We follow idiomatic Go standards to ensure code quality and consistency.

#### Formatting

- Use `gofmt` for all Go code (automatically enforced)
- Run before committing:
  ```bash
  gofmt -w .
  ```

#### Naming Conventions

- Use short, descriptive variable names (`user` instead of `u`, `users` instead of `userList`)
- Package names are lowercase, single words when possible
- Exported symbols (functions, types, constants) start with uppercase
- Use `err` for error variables (never `e` or `error`)
- Use `i`, `j`, `k` only for loop indices

#### Function and Method Design

```go
// Good: Clear, descriptive names
func (u *User) Validate() error { ... }
func NewUserService(repo UserRepository) *UserService { ... }

// Good: Method receivers use short names
func (us *UserService) CreateUser(ctx context.Context, u *User) error { ... }

// Avoid: Generic or unclear names
func (u *User) Do() { ... }
func Process() { ... }
```

#### Error Handling

Always handle errors explicitly:

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// Good: Use custom domain errors
var ErrUserNotFound = errors.New("user not found")
if errors.Is(err, sql.ErrNoRows) {
    return ErrUserNotFound
}

// Never: Ignore errors
_ = database.Save(user)  // Bad!

// Never: Use panic for expected errors
panic(err)  // Bad!
```

#### Architecture

Follow Clean Architecture with clear layer separation:

```
┌─────────────────┐
│  Handlers       │  HTTP/gRPC entry points
├─────────────────┤
│  Services       │  Business logic
├─────────────────┤
│  Repositories   │  Data access
├─────────────────┤
│  Domain         │  Models & interfaces
└─────────────────┘
```

Pattern: **Handlers → Services → Repositories**

```go
// Domain: Define interfaces
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}

// Service: Business logic depending on interface
type UserService struct {
    repo UserRepository
    log  *log.Logger
}

func NewUserService(repo UserRepository, log *log.Logger) *UserService {
    return &UserService{repo: repo, log: log}
}

func (us *UserService) CreateUser(ctx context.Context, name string) error {
    user := &User{ID: uuid.New().String(), Name: name}
    if err := user.Validate(); err != nil {
        return err
    }
    return us.repo.Save(ctx, user)
}

// Handler: HTTP layer
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.service.CreateUser(r.Context(), req.Name); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}
```

#### Dependency Injection

Use constructor functions with interfaces (never global state):

```go
// Good: Explicit dependencies
func NewAPIHandler(svc UserService, log *log.Logger) *APIHandler {
    return &APIHandler{
        svc: svc,  // Interface, not concrete type
        log: log,
    }
}

// Bad: Global state
var globalDB *sql.DB
var globalLogger *log.Logger
```

#### Comments

Follow Go comment conventions:

```go
// Package user provides user management functionality.
package user

// User represents a system user with authentication.
type User struct {
    ID    string // Unique identifier
    Email string // Primary email address
    Name  string // Display name
}

// NewUser creates a new User with the given email and name.
// Returns an error if email is invalid or name is empty.
func NewUser(email, name string) (*User, error) {
    // ...
}
```

### Code Quality Standards

- **Linting:** Must pass `golangci-lint run` with zero errors
- **Test Coverage:** Minimum 80% on critical paths (business logic, handlers)
- **Cyclomatic Complexity:** Keep functions under 15 lines of decision paths
- **Documentation:** GoDoc comments on all exported symbols
- **No Dead Code:** Remove unused functions, variables, and imports

Run checks before committing:

```bash
gofmt -w .
golangci-lint run
go test -cover ./...
```

## Testing Requirements

### Unit Tests

Test each layer independently with mocked dependencies:

```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        input   *User
        want    error
        wantErr bool
    }{
        {"valid user", &User{Name: "John", Email: "john@example.com"}, nil, false},
        {"empty name", &User{Name: "", Email: "john@example.com"}, ErrInvalidInput, true},
        {"invalid email", &User{Name: "John", Email: "invalid"}, ErrInvalidEmail, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &MockUserRepository{}
            svc := NewUserService(repo, log.NewLogger())

            err := svc.CreateUser(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
            if err != tt.want && !errors.Is(err, tt.want) {
                t.Errorf("CreateUser() error = %v, want %v", err, tt.want)
            }
        })
    }
}
```

### Integration Tests

Test interactions between real components:

```go
// Use build tags for integration tests
//go:build integration

func TestUserService_Integration(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := repository.NewUserRepository(db)
    svc := NewUserService(repo, log.NewLogger())

    user := &User{Name: "John", Email: "john@example.com"}
    if err := svc.CreateUser(context.Background(), user); err != nil {
        t.Fatalf("CreateUser() failed: %v", err)
    }

    retrieved, err := repo.GetByEmail(context.Background(), "john@example.com")
    if err != nil {
        t.Fatalf("GetByEmail() failed: %v", err)
    }
    if retrieved.Name != "John" {
        t.Errorf("Name = %q, want %q", retrieved.Name, "John")
    }
}
```

### Test Organization

```
pkg/
  user/
    user.go
    user_test.go              # Unit tests
    user_integration_test.go  # Integration tests
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Unit tests only
go test -short ./...

# Integration tests only
go test -tags=integration ./...

# Using Makefile
make test
make test-race
make test-coverage
```

### Test Naming

Test functions follow Go convention:

```go
// TestPackageName_FunctionName_Scenario
func TestUserService_CreateUser_ValidInput(t *testing.T) { ... }
func TestUserService_CreateUser_InvalidEmail(t *testing.T) { ... }
func TestUserRepository_GetByID_NotFound(t *testing.T) { ... }
```

## Pull Request Process

### Before Creating a PR

1. **Check Beads for ready tasks:**
   ```bash
   bd ready --json
   ```

2. **Create or claim a task:**
   ```bash
   bd create "Brief description of work" -t feature
   # or
   /task-start bd-xxxx  # Start existing task
   ```

3. **Reserve files you'll modify:**
   ```bash
   /reserve pkg/myfeature/**/*.go
   ```

4. **Create a branch from `main`:**
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/bd-xxxx-short-description
   ```

5. **Make your changes:**
   - Write code following our style guide
   - Write tests alongside your code
   - Run tests locally: `go test ./...`

6. **Format and lint:**
   ```bash
   gofmt -w .
   golangci-lint run
   ```

### Creating the PR

1. **Push your branch:**
   ```bash
   git push origin feature/bd-xxxx-short-description
   ```

2. **Create PR with clear description:**
   - Title: `[bd-xxxx] Brief one-line summary`
   - Body: Explain what and why, reference related issues
   - Link to Beads task in description

3. **Request review:**
   ```bash
   /coordinate reviewer "Review of feature bd-xxxx at <branch-name>"
   ```

### PR Checklist

Before submitting:
- [ ] Tests pass locally (`go test ./...`)
- [ ] Code is formatted (`gofmt -w .`)
- [ ] Linting passes (`golangci-lint run`)
- [ ] Coverage maintained or improved (min 80% critical paths)
- [ ] Documentation updated (comments, README, etc.)
- [ ] No breaking changes (or documented migration path)
- [ ] Commit messages are clear and descriptive
- [ ] Beads task status updated

### PR Review Process

1. **Code review agent examines:**
   - Code style and Go idioms
   - Test coverage and quality
   - Architecture and design patterns
   - Potential bugs and edge cases
   - Performance considerations

2. **Author addresses feedback:**
   - Push new commits (don't force-push)
   - Reply to comments with explanations
   - Update Beads task status as needed

3. **Approval and merge:**
   - Reviewer approves when satisfied
   - Use squash merge for clean history
   - Reference Beads task in merge commit: `Closes bd-xxxx`

## Beads Workflow for Contributors

[Beads](https://github.com/steveyegge/beads) is our Git-native issue tracking system. All work flows through Beads tasks.

### Task Lifecycle

```
┌─────────────┐
│   ready     │  Unblocked, available for work
└──────┬──────┘
       │ /task-start
       ▼
┌─────────────┐
│ in_progress │  Currently being worked on
└──────┬──────┘
       │ /task-complete
       ▼
┌─────────────┐
│    done     │  Completed
└─────────────┘
```

### Creating Tasks

```bash
# Simple task
bd create "Implement user authentication"

# With priority and type
bd create "Fix null pointer in handler" -t bug -p high

# With parent task (subtask)
bd create "Add JWT validation" --parent bd-a1b2

# With tags
bd create "Update database schema" -t feature --tags database,migrations
```

### Task Status Updates

```bash
# Start task (from ready)
/task-start bd-xxxx

# During work, update status
bd update bd-xxxx --status in_progress

# File discovered issues
bd create "Found race condition" --parent bd-xxxx --type discovered

# Complete task
/task-complete bd-xxxx
# or
bd close bd-xxxx --reason "Implemented and tested"
```

### Managing Dependencies

Link tasks when one blocks another:

```bash
# Task B depends on Task A being done first
bd dep add bd-B bd-A --type blocks
# (bd-B is blocked by bd-A)

# View dependencies
bd show bd-xxxx
```

### Querying Tasks

```bash
# View ready (unblocked) tasks
bd ready --json

# List all tasks
bd list

# List with filters
bd list --status in_progress
bd list --type bug
bd list --priority high

# View specific task
bd show bd-xxxx
```

### Task-Message Linking

Link messages to tasks for traceability:

```bash
# Send message with task reference
/coordinate agent-name "Task bd-xxxx implementation complete"
# Include thread_id: bd-xxxx for proper linking
```

## Development Setup

### Prerequisites

Install these tools in order:

1. **Go 1.25+**
   ```bash
   go version  # Should show go1.25.x or higher
   ```

2. **OpenCode (SST)**
   ```bash
   curl -fsSL https://opencode.ai/install | bash
   opencode --version
   ```

3. **Agent Mail MCP Server**
   ```bash
   curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh?$(date +%s)" | bash -s -- --yes
   am  # Test it (Ctrl+C to stop)
   ```

4. **Beads**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
   bd --version
   ```

5. **Serena**
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   uvx --from git+https://github.com/oraios/serena serena --help
   ```

6. **Development Tools**
   ```bash
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

### Project Setup

```bash
# Clone or navigate to project
cd /home/lewis/src/open-swarm

# Install Go dependencies
go mod download

# Initialize Beads (if first time)
bd init

# Start Agent Mail server (keep running in separate terminal)
am

# Build the project
go build -o bin/open-swarm ./cmd/open-swarm

# Run tests
go test ./...

# Start OpenCode
opencode
```

### Environment Variables

Create `.env` file in project root (not committed):

```bash
# Required
ANTHROPIC_API_KEY=sk-ant-...

# Optional
OPENAI_API_KEY=sk-...
GOOGLE_GENERATIVE_AI_API_KEY=...
```

Or set temporarily:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
opencode
```

### Useful Makefile Targets

```bash
# Build all binaries
make build

# Run tests with coverage
make test

# Format code
make fmt

# Run tests with race detector
make test-race

# Generate coverage HTML report
make test-coverage

# Start Docker services (Temporal + PostgreSQL)
make docker-up

# Stop Docker services
make docker-down

# View Docker logs
make docker-logs

# Clean up binaries
make clean
```

## Adding New Workflows

Workflows define multi-step processes for agents to follow. They're stored in `.opencode/command/` as markdown files.

### Workflow Structure

Workflows are markdown files with YAML frontmatter:

```yaml
---
description: One-line description of what this workflow does
---

## Step 1: First Step

Instructions here...

## Step 2: Second Step

More instructions...
```

### Example: Creating a Database Migration Workflow

Create `.opencode/command/db-migrate.md`:

```markdown
---
description: Create and apply a database migration
---

# Database Migration Workflow

Execute the database migration process:

## 1. Create Migration File

Run:
```bash
cd migrations
go run . create -name "<migration_name>"
```

## 2. Implement Migration

Edit the newly created migration file in `migrations/migrations/`.

Add `Up()` implementation for schema changes.
Add `Down()` implementation to revert changes.

## 3. Test Migration

Run locally:
```bash
go run ./migrations . up
```

Verify with:
```bash
go run ./migrations . status
```

## 4. Add Regression Test

Create test in `pkg/migrations/migrations_test.go`:

```go
func TestMigration_<Name>(t *testing.T) {
    // Test implementation
}
```

## 5. Document Changes

Update `docs/DATABASE.md` with:
- Migration purpose
- Schema changes
- Rollback procedure

## 6. Update Beads Task

When complete:
```bash
bd update bd-xxxx --status done
```
```

### Registering Workflows

Workflows in `.opencode/command/` are automatically available as `/workflow-name` commands in OpenCode.

### Workflow Best Practices

- Keep workflows focused on single processes
- Include clear, step-by-step instructions
- Provide example commands
- Reference Beads tasks where applicable
- Document expected outcomes
- Include troubleshooting tips if complex

## Adding New Activities

Activities are operations that agents can perform. Customize the workflow by creating new activities in `.opencode/tool/`.

### Activity Structure

Activities are TypeScript/JavaScript files that OpenCode can execute:

```typescript
// .opencode/tool/custom-activity.ts
import { ActivityHandler } from "opencode-sdk";

export const customActivity: ActivityHandler = {
  name: "custom_activity",
  description: "Performs a custom operation",

  async handle(input: {
    param1: string;
    param2: number;
  }): Promise<void> {
    // Implementation
  }
};
```

### Example: Creating a Beads Sync Activity

Create `.opencode/tool/beads-sync.ts`:

```typescript
import { ActivityHandler, Shell } from "opencode-sdk";

export const beadsSyncActivity: ActivityHandler = {
  name: "beads_sync",
  description: "Sync Beads tasks and update Git",

  async handle(input: {
    taskId?: string;
    status?: "ready" | "in_progress" | "done" | "blocked";
  }): Promise<{
    synced: boolean;
    taskStatus: string;
  }> {
    const shell = new Shell();

    // Get current Beads status
    const { stdout } = await shell.exec("bd list --json");
    const tasks = JSON.parse(stdout);

    if (input.taskId) {
      // Update specific task
      await shell.exec(`bd update ${input.taskId} --status ${input.status}`);
    }

    // Sync to Git
    await shell.exec("git add .beads/issues.jsonl");
    await shell.exec('git commit -m "Update Beads tasks"');

    return {
      synced: true,
      taskStatus: input.status || "synced"
    };
  }
};
```

### Activity Best Practices

- Keep activities focused and single-purpose
- Use OpenCode SDK for reliability
- Handle errors gracefully
- Log important operations
- Document input/output schemas
- Make activities testable

### Registering Activities

Activities are automatically discovered from `.opencode/tool/` and available in agents' capabilities.

## Multi-Agent Coordination

When multiple agents work on the same codebase:

### Session Start (Every Agent)

```bash
# 1. In separate terminal, start Agent Mail
am

# 2. Start OpenCode
opencode

# 3. Run session protocol
/session-start
```

### Parallel Development Pattern

```
Agent A: Backend (reserves internal/api/**)
         ↓ [Implements API endpoints]
         └─→ Sends message: "API ready"

Agent B: Frontend (reserves web/src/**)
         ↓ [Waits for Agent A notification]
         ↓ [Integrates with API]
         └─→ Replies: "Integration complete"

Agent C: Tests (reserves **/*_test.go)
         ↓ [Waits for implementations]
         ↓ [Writes integration tests]
         └─→ Reports: "All tests passing"
```

### File Reservation Etiquette

- Always reserve files before editing: `/reserve pkg/myfeature/**/*.go`
- Use specific patterns, not broad globs (`**/*.go` causes conflicts!)
- Release when done: `/release`
- Renew if work takes longer than 1 hour: Use Agent Mail's renew function
- Check reservations: Query Agent Mail for active reservations

### Communication Protocol

1. **Blocking Work Completion:**
   ```
   To: NextAgent
   Subject: [bd-xxxx] API endpoints ready
   Thread: bd-xxxx
   Body: Implemented CRUD endpoints at /api/v1/users. Docs at /docs/api/users.md
   ```

2. **Help Requests:**
   ```
   To: SpecialistAgent
   Subject: Help needed: database migration race condition
   Importance: high
   Body: Encountering deadlock in foreign key migration. Review needed.
   ```

3. **Status Updates:**
   ```
   To: All agents
   Subject: [bd-yyyy] Feature complete
   Thread: bd-yyyy
   Body: Authentication system ready. All agents can use /api/auth endpoints.
   ```

### Conflict Resolution

When agents need the same files:

1. **Check who has reservation:**
   ```bash
   # Query Agent Mail for active reservations
   ```

2. **Coordinate via message:**
   ```bash
   /coordinate HoldingAgent "Need to edit file X, ETA on release?"
   ```

3. **Resolve:**
   - Wait for file release (1 hour TTL default)
   - Work on different files
   - Agree on sequential access

## Troubleshooting

### Agent Mail Not Connecting

```bash
# Check if server is running
curl http://localhost:8765/health

# Start server
am

# Check logs
tail -f ~/.agent-mail/server.log

# Restart
pkill -f "mcp_agent_mail"
am
```

### OpenCode Can't Find MCP Servers

```bash
# Verify Agent Mail works
python -m mcp_agent_mail.server &

# Verify Serena works
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --cwd . &

# Check opencode.json
cat opencode.json | grep -A 10 '"mcp"'
```

### Beads Sync Issues

```bash
# Force sync
bd sync

# Check status
bd list --json

# Re-initialize if corrupt
rm -rf .beads
bd init
```

### File Reservation Conflicts

```bash
# Check who holds reservation
# (via Agent Mail queries)

# Release stale reservation
# Option 1: Wait for TTL (1 hour)
# Option 2: Message the agent
# Option 3: Work on different files
```

### Tests Failing

```bash
# Run with verbose output
go test -v ./pkg/mypackage

# Run specific test
go test -run TestMyFunction ./pkg/mypackage

# Run with race detector
go test -race ./...

# Check coverage on failing test
go test -cover ./pkg/mypackage
```

### Build Issues

```bash
# Clean build
go clean
go build -o bin/open-swarm ./cmd/open-swarm

# Check dependencies
go mod download
go mod verify

# Update dependencies
go get -u ./...
go mod tidy
```

## Getting Help

- **Project Guide:** See `AGENTS.md` for comprehensive multi-agent workflows
- **Quick Start:** See `QUICKSTART.md` for first-time setup
- **Beads Guide:** See `.beads/README.md` for issue tracking
- **Architecture:** See `README.md` for project structure
- **Workflows:** See `docs/` for specific workflows (TCR, DAG, Monitoring)
- **Examples:** See `examples/` for example tasks and workflows

## Code of Conduct

- Be respectful to other agents and contributors
- Communicate clearly and proactively
- Release file reservations promptly
- Update Beads tasks regularly
- Document your changes
- Help other agents when possible

Happy coding! 🚀
