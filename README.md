# Open Swarm

A Temporal-based workflow orchestration system for OpenCode AI coding agents, using [OpenCode](https://opencode.ai) (from SST), [Agent Mail](https://github.com/Dicklesworthstone/mcp_agent_mail), [Beads](https://github.com/steveyegge/beads), and [Serena](https://github.com/oraios/serena).

## Overview

Open Swarm orchestrates multiple AI coding agents using Temporal workflows, with each agent running in an isolated environment (dedicated port + git worktree + OpenCode server instance). The system ensures agents can work in parallel without conflicts through:

- **Temporal Workflows** - TCR (Test-Commit-Revert) and DAG execution patterns
- **Agent Mail MCP Server** - Git-backed messaging and advisory file reservations
- **Beads MCP** - Lightweight, distributed issue tracking with dependency management
- **Serena MCP Server** - LSP-powered semantic code navigation
- **OpenCode SDK** - Provider-agnostic AI coding agent integration

## Features

### Workflow Orchestration

- **TCR Workflow** - Test-Commit-Revert pattern for safe iterative development
- **DAG Workflow** - Dependency-aware task execution with parallel processing
- **Enhanced TCR** - Extended workflow with validation gates and multi-review
- **Cell Isolation** - Each agent gets isolated port (8000-9000), git worktree, and server instance

### Multi-Agent Coordination

- **Cooperative Conflict Resolution** - Negotiation-first approach via Agent Mail
- **File Reservations** - Advisory locks with expiration and renewal
- **Real-time Logging** - Structured logging (slog) of all agent interactions
- **Message Threading** - Organized conversations linked to tasks
- **Task Dependencies** - Track blocking relationships via Beads

### Developer Experience

- **Session Protocols** - Standardized start/end workflows via slash commands
- **MCP Integration** - 7 MCP servers pre-configured (Agent Mail, Beads, Serena, etc.)
- **TDD Enforcement** - Optional TDD Guard plugin for test-first development
- **Comprehensive Testing** - Unit, integration, and E2E tests

## Quick Start

### Prerequisites

Install all required tools:

```bash
# OpenCode (SST)
curl -fsSL https://opencode.ai/install | bash

# Agent Mail
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh?$(date +%s)" | bash -s -- --yes

# Beads
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Serena
pip install uv  # or download from https://docs.astral.sh/uv/
# Serena runs via uvx on demand

# Temporal (for workflow execution)
# See: https://docs.temporal.io/cli#install

# Go 1.25+
# (varies by OS - see https://go.dev/dl/)

# Verify installations
opencode --version
am  # Should start Agent Mail server
bd --version
temporal --version
go version
```

### Project Setup

```bash
# Clone or navigate to project
cd open-swarm

# Install Go dependencies
go mod download

# Initialize Beads (if not already done)
bd init

# Build all binaries
make build

# Test
make test
```

### Running Workflows

```bash
# Terminal 1: Start Temporal server
docker-compose up

# Terminal 2: Start Agent Mail
am

# Terminal 3: Start Temporal worker
go run ./cmd/temporal-worker

# Terminal 4: Submit a workflow
go run ./cmd/reactor-client \
  --workflow tcr \
  --task-id "implement-auth" \
  --prompt "Add JWT authentication to /api/login"
```

### Using OpenCode Directly

```bash
# 1. Start Agent Mail server (in separate terminal)
am

# 2. Start OpenCode
opencode

# 3. Run session start protocol
/session-start

# 4. Start working!
# OpenCode will guide you through:
# - Agent registration
# - Checking inbox
# - Viewing ready tasks
# - Reserving files
# - Coordinating with other agents
```

## Configuration

### opencode.json

Main configuration file with:
- **MCP Servers:** 7 servers configured (Agent Mail, Beads, Serena, Sequential Thinking, Chrome DevTools, OpenCode Agent, Playwright)
- **Custom Agents:** `validator`, `coordinator`, `reviewer`, `tester`, `explorer`, `documenter`
- **Custom Commands:** `/validate`, `/sync`, `/reserve`, `/release`, `/task-start`, `/task-complete`, `/task-ready`, `/coordinate`, `/review`, `/explore`, `/test`, `/doc`, `/build`, `/lint`
- **Formatters:** Auto-format Go code with `gofmt`
- **Permissions:** Granular control over tool usage

See `opencode.json` for full configuration.

### AGENTS.md

Project-specific instructions for AI agents including:
- Architecture overview
- Development workflows
- Code standards
- Multi-agent coordination patterns
- Testing strategy
- **CRITICAL RULES:** Beads mandatory, Serena for code editing, TDD required

Read `AGENTS.md` for complete details.

## Usage

### Core Slash Commands

| Command | Description |
|---------|-------------|
| `/session-start` | Complete session initialization protocol |
| `/session-end` | Clean session termination with handoff |
| `/sync` | Sync with Agent Mail (register + fetch inbox) |
| `/reserve <pattern>` | Reserve files for exclusive editing |
| `/release` | Release all file reservations |
| `/task-start <id>` | Start Beads task and reserve files |
| `/task-complete <id>` | Complete task and release files |
| `/task-ready` | Check Beads for unblocked work |
| `/coordinate <agent> <subject>` | Send coordination message |
| `/review <files>` | Code review with specialized agent |
| `/validate` | Deep validation using Claude Opus 4.5 |
| `/explore` | Fast codebase exploration with Haiku |

### Beads Integration (via MCP)

Beads functionality is provided by the **Beads MCP server** (`beads-mcp`), configured in `opencode.json`. Access via OpenCode MCP tools:

```bash
# In OpenCode, these tools are available:
beads_ready           # Get unblocked tasks
beads_status          # Update task status
beads_close           # Close task with reason
beads_create          # Create new task
beads_list            # List tasks with filters
beads_addDependency   # Link tasks

# Or use bd CLI directly:
bd ready --json
bd update <task-id> --status in_progress
bd close <task-id> --reason "Completed"
```

## Monitoring & Logging

Open Swarm uses Go's `log/slog` for comprehensive structured logging of all agent interactions.

### View Logs in Real-Time

```bash
# Run interactive demo to see agent coordination
go run ./cmd/logging-demo

# JSON format for production monitoring
LOG_FORMAT=json go run ./cmd/logging-demo
```

### What Gets Logged

#### Agent Lifecycle
```
INFO: New agent registered (name, program, model, task, project)
INFO: Active agents in project (count)
INFO: Agent updated (previous_task → new_task)
INFO: Agent removed from project (remaining_agents)
```

#### Conflict Detection & Resolution
```
INFO: Checking for conflicts (agent, pattern, exclusive)
WARN: Exclusive conflict detected (requestor, holder, expires_at)
ERROR: CONFLICT DETECTED (conflict_type, num_conflicts)
INFO: Resolution: NEGOTIATE - contact holders via Agent Mail
INFO: Resolution: WAIT - all reservations expire soon
WARN: Resolution: FORCE RELEASE - stale reservations detected
```

#### Coordination
```
INFO: Starting coordination sync (active_agents)
INFO: Project registered in Agent Mail
INFO: Message queue checked
INFO: File reservations updated
INFO: Coordination sync complete
```

### Conflict Resolution Philosophy

Open Swarm uses a **cooperative, non-hostile** approach to conflict resolution:

1. **Negotiate** (Default) - Contact holders via Agent Mail to coordinate
2. **Wait** (Polite) - Wait for reservations to expire (≤5 minutes)
3. **Force Release** (Responsible) - Only for stale (already expired) locks
4. **Change Pattern** (Accommodating) - Modify file pattern to avoid overlap

**No aggressive retries, no blame assignment, no competitive behavior.**

## Architecture

```
┌─────────────────────────────────────────┐
│       Temporal Workflow Engine          │
│   (TCR, DAG, Enhanced TCR patterns)     │
└────────────┬────────────────────────────┘
             │
       ┌─────┴─────┬──────────┬────────┐
       ▼           ▼          ▼        ▼
   ┌────────┐  ┌───────┐  ┌──────┐  ┌────┐
   │OpenCode│  │  Git  │  │ MCP  │  │Port│
   │  SDK   │  │Worktree│ │Servers│ │Mgr │
   │Clients │  │(Isolated│ │(7 of │  │8000│
   │Per-Cell│  │Branches)│ │them) │  │9000│
   └────────┘  └───────┘  └──────┘  └────┘
```

**How It Works:**
1. Temporal workflows orchestrate agent execution
2. Each workflow spawns "cells" (isolated agent environments)
3. Each cell gets: dedicated port + git worktree + OpenCode server
4. TCR pattern: Test → (Pass=Commit, Fail=Revert)
5. DAG workflows: Execute dependent tasks in topological order
6. MCP servers handle coordination, issue tracking, and code navigation

## Project Structure

```
open-swarm/
├── opencode.json          # OpenCode configuration
├── AGENTS.md             # Agent instructions (CRITICAL RULES)
├── AUDIT-FINDINGS.md     # Code audit & documentation review
├── cmd/
│   ├── temporal-worker/  # Temporal workflow worker
│   ├── reactor-client/   # Workflow submission client
│   ├── single-agent-demo/# OpenCode SDK demo
│   ├── workflow-demo/    # Workflow visualization
│   ├── logging-demo/     # Agent coordination logging demo
│   ├── quality-monitor/  # Quality monitoring tool
│   └── agent-automation-demo/ # Automation demo
├── internal/             # Core implementation (87% of code)
│   ├── temporal/         # Temporal workflows (TCR, DAG, Enhanced)
│   ├── infra/            # Infrastructure (ports, servers, worktrees)
│   ├── agent/            # OpenCode SDK client
│   ├── workflow/         # Workflow activities
│   ├── conflict/         # Conflict analyzer with cooperative resolution
│   ├── mergequeue/       # Speculative merge queue (in progress)
│   └── config/           # Configuration
├── pkg/                  # Public packages (13% of code)
│   ├── coordinator/      # Coordination helpers (placeholder)
│   └── agent/            # Agent management (in-memory tracking)
├── test/                 # Integration tests
├── docs/                 # Documentation
│   ├── ARCHITECTURE.md
│   ├── TCR-WORKFLOW.md
│   ├── DAG-WORKFLOW.md
│   ├── MONITORING.md
│   ├── DEPLOYMENT.md
│   ├── SECURITY.md
│   └── ...
└── Makefile             # Build and test targets
```

**Note:** The majority of the implementation is in `internal/`, not `pkg/`. This is intentional - `internal/` contains the production-quality Temporal workflows and infrastructure, while `pkg/` provides minimal public APIs.

## Development

### Running Tests

```bash
# All tests
make test

# With race detector
make test-race

# Coverage report (HTML)
make test-coverage

# With TDD Guard reporter
make test-tdd

# Direct go test
go test ./...
go test -cover ./...
```

#### TDD Guard

This project optionally uses [TDD Guard](https://github.com/nizos/tdd-guard) for enhanced test reporting.

**Installation:**
```bash
go install github.com/nizos/tdd-guard/reporters/go/cmd/tdd-guard-go@latest
```

**Usage:**
```bash
make test-tdd
# Or directly:
go test -json ./... 2>&1 | tdd-guard-go -project-root $(pwd)
```

### Building

```bash
# Build all binaries
make build

# Individual binaries
go build -o bin/temporal-worker ./cmd/temporal-worker
go build -o bin/reactor-client ./cmd/reactor-client
go build -o bin/single-agent-demo ./cmd/single-agent-demo
go build -o bin/workflow-demo ./cmd/workflow-demo
go build -o bin/logging-demo ./cmd/logging-demo

# Production (optimized)
go build -ldflags="-s -w" -o bin/temporal-worker ./cmd/temporal-worker
```

**Note:** There is no `cmd/open-swarm/` directory. Use the binaries above.

### Code Quality

```bash
# Format
make fmt
# Or: gofmt -w .

# Lint - Easy interface (recommended)
./scripts/lint            # Run check (default)
./scripts/lint fix        # Auto-fix issues
./scripts/lint progress   # Show dashboard
./scripts/lint autofix    # Smart auto-fix
./scripts/lint --help     # See all options

# Lint - Traditional targets
make lint                 # Run check
make lint-fix            # Auto-fix

# All CI checks
make ci
```

See [docs/linting.md](docs/linting.md) for detailed linting documentation.

## Workflow Patterns

### TCR (Test-Commit-Revert)

Traditional Test-Commit-Revert pattern for safe iterative development:

1. **Bootstrap** - Allocate port, create worktree, start OpenCode server
2. **Execute** - Run agent prompt to make changes
3. **Test** - Run test suite
4. **Commit or Revert** - If tests pass → commit, else → revert
5. **Teardown** - Clean up resources (saga pattern)

```bash
go run ./cmd/reactor-client \
  --workflow tcr \
  --task-id "fix-bug-123" \
  --prompt "Fix the null pointer in handleRequest"
```

### DAG (Directed Acyclic Graph)

Execute tasks in topological order with dependency management:

1. **Define tasks** with dependencies in YAML
2. **Toposort** to determine execution order
3. **Execute** tasks in order, parallelizing independent tasks
4. **Retry loop** on failure with human intervention signal

```yaml
# tasks.yaml
tasks:
  - id: setup-db
    command: "go run ./scripts/setup-db.go"
  - id: run-migrations
    command: "go run ./cmd/migrate"
    depends_on: [setup-db]
  - id: run-tests
    command: "go test ./..."
    depends_on: [run-migrations]
```

```bash
go run ./cmd/reactor-client --workflow dag --tasks tasks.yaml
```

## Multi-Agent Workflows

### Parallel Development

Multiple agents work on independent features simultaneously with isolated environments:

```
Agent A (Cell 1 - Port 8000): Backend API (reserves internal/api/**)
Agent B (Cell 2 - Port 8001): Frontend (reserves web/**)
Agent C (Cell 3 - Port 8002): Tests (reserves test/**)

Each runs in isolated git worktree, preventing conflicts by design.
```

### Cooperative Conflict Resolution

When file patterns overlap:

```
Agent A: Requests internal/auth/*.go (exclusive)
Agent B: Already holds internal/auth/*.go (exclusive)

System Response:
  1. Detects conflict via pattern matching
  2. Logs: WARN "Exclusive conflict detected"
  3. Suggests: "NEGOTIATE - contact holders via Agent Mail"
  4. Agent A sends message: "/coordinate Agent B 'Need to refactor auth, can we sync?'"
  5. Agents coordinate and adjust reservations
```

**No forced takeovers, no aggressive retries.**

### Sequential Pipeline

Work flows through stages handled by different agents:

```
Agent A: Schema Design → /coordinate Agent B "Schema ready"
    ↓
Agent B: Migrations → /coordinate Agent C "Migrations ready"
    ↓
Agent C: Testing → Complete task
```

## Best Practices

### File Reservations

- **Always reserve before editing** - Use `/reserve <pattern>`
- **Be specific** - Use narrow patterns like `internal/auth/*.go`, not `**/*`
- **Release promptly** - Run `/release` when done
- **Renew if needed** - Extend TTL if work takes >1 hour
- **Check expiration** - Logs show when reservations expire

### Task Management

- **Break work into small tasks** - Easier to track and coordinate
- **Update Beads frequently** - Keep status current
- **Use dependencies** - Link tasks with `bd dep add`
- **Close with reasons** - Document what was done

### Communication

- **Clear subjects** - Include task IDs: `[bd-a1b2] Auth complete`
- **Set importance** - Use `normal` for routine, `high` for urgent
- **Require acks** - For critical coordination messages
- **Coordinate early** - Don't wait until conflicts occur

### Session Hygiene

- **Always `/session-start`** - Registers agent, checks inbox
- **Always `/session-end`** - Releases files, syncs state
- **Never leave reservations** - Auto-expire but clean up manually
- **Sync Beads to Git** - `bd sync` at end of session

## Troubleshooting

### Agent Mail Not Connecting

```bash
# Check if server is running
curl http://localhost:8765/health

# Restart server
am

# Check logs
tail -f ~/.agent-mail/server.log
```

### Temporal Not Running

```bash
# Start Temporal via Docker
docker-compose up

# Or install and run locally
temporal server start-dev

# Check status
temporal workflow list
```

### OpenCode Not Finding MCP Servers

```bash
# Verify MCP servers can start
python -m mcp_agent_mail.server &
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --cwd . &

# Check opencode.json MCP configuration
cat opencode.json | jq '.mcp'
```

### Beads Sync Issues

```bash
# Force sync
bd sync

# Check integrity
bd list --json

# Re-initialize if corrupt
bd init
```

### View Logs for Debugging

```bash
# Run logging demo to see what's happening
go run ./cmd/logging-demo

# Enable DEBUG level logging
LOG_LEVEL=DEBUG go run ./cmd/temporal-worker

# JSON logs for parsing
LOG_FORMAT=json go run ./cmd/temporal-worker
```

## Resources

- [OpenCode Documentation](https://opencode.ai/docs/)
- [Agent Mail GitHub](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [Beads GitHub](https://github.com/steveyegge/beads)
- [Serena Documentation](https://oraios.github.io/serena/)
- [Temporal Documentation](https://docs.temporal.io/)
- [AGENTS.md Format](https://agents.md)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [AUDIT-FINDINGS.md](AUDIT-FINDINGS.md) - Comprehensive code audit

## License

MIT License - See LICENSE file for details

## Contributing

1. Check available work: `bd ready --json`
2. Start task: `bd update <task-id> --status in_progress`
3. Reserve files: `/reserve <pattern>` (in OpenCode)
4. Make changes following AGENTS.md guidelines
5. Run tests: `make test`
6. Complete task: `bd close <task-id> --reason "Description"`
7. Sync: `bd sync`

**Before contributing, read:**
- `AGENTS.md` - Critical rules (Beads mandatory, Serena for code, TDD required)
- `AUDIT-FINDINGS.md` - Current state of the codebase
- `docs/ARCHITECTURE.md` - System design and invariants

## Support

For issues:
- Check `AGENTS.md` for detailed workflows
- Review `AUDIT-FINDINGS.md` for known discrepancies
- Check MCP server status
- View logs: `go run ./cmd/logging-demo`
- File issue in Beads: `bd create --title="Issue description" --type=bug`

## What Changed Recently

### 2025-12-13: Comprehensive Logging Added

- Added structured logging (slog) to all agent coordination paths
- Created `cmd/logging-demo/` to visualize agent interactions
- Verified cooperative (non-hostile) conflict resolution
- Created `AUDIT-FINDINGS.md` with complete code audit

**Run the demo:** `go run ./cmd/logging-demo`

### Documentation Updated

- README now accurately reflects Temporal-based architecture
- Removed references to non-existent `cmd/open-swarm/`
- Corrected build instructions
- Added monitoring & logging section
- Clarified MCP server integration (not custom tools)
