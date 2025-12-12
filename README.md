# Open Swarm

A multi-agent coordination framework for Go projects using [OpenCode](https://opencode.ai) (from SST), [Agent Mail](https://github.com/Dicklesworthstone/mcp_agent_mail), [Beads](https://github.com/steveyegge/beads), and [Serena](https://github.com/oraios/serena).

## Overview

Open Swarm enables multiple AI coding agents to collaborate on the same codebase without conflicts through:

- **Agent Mail MCP Server** - Git-backed messaging and advisory file reservations
- **Beads** - Lightweight, distributed issue tracking with dependency management
- **Serena MCP Server** - LSP-powered semantic code navigation
- **OpenCode** - Provider-agnostic AI coding agent platform (SST)

## Features

### Multi-Agent Coordination

- **Agent Registration** - Unique identity per agent with memorable names (e.g., "BlueLake")
- **Message Threading** - Organized conversations linked to tasks
- **File Reservations** - Advisory locks prevent edit conflicts
- **Task Dependencies** - Track blocking relationships between work items
- **Semantic Navigation** - LSP-powered code exploration via Serena

### Developer Experience

- **Session Protocols** - Standardized start/end workflows
- **Custom Commands** - Slash commands for common coordination tasks
- **Custom Tools** - Beads integration tools for task management
- **Custom Agents** - Specialized agents for review, testing, coordination

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

# Go 1.25+
# (varies by OS - see https://go.dev/dl/)

# Verify installations
opencode --version
am  # Should start Agent Mail server
bd --version
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

# Build the CLI
go build -o bin/open-swarm ./cmd/open-swarm

# Test
go test ./...
```

### Start Coding

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
- **MCP Servers:** Agent Mail and Serena auto-configured
- **Custom Agents:** `coordinator`, `reviewer`, `tester`
- **Custom Commands:** `/sync`, `/reserve`, `/release`, `/task-start`, etc.
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

### Custom Tools

Beads integration tools available to agents:

- `beads.ready` - Get unblocked tasks
- `beads.status` - Update task status
- `beads.close` - Close task with reason
- `beads.create` - Create new task
- `beads.list` - List tasks with filters
- `beads.addDependency` - Link tasks

## Architecture

```
┌─────────────────────────────────────────┐
│           OpenCode (SST)                │
│  Provider-agnostic AI coding platform   │
└────────────┬────────────────────────────┘
             │
       ┌─────┴─────┬──────────┬────────┐
       ▼           ▼          ▼        ▼
   ┌────────┐  ┌───────┐  ┌──────┐  ┌────┐
   │ Agent  │  │ Beads │  │Serena│  │ Go │
   │ Mail   │  │       │  │      │  │1.25│
   │  MCP   │  │  CLI  │  │ MCP  │  │    │
   └────────┘  └───────┘  └──────┘  └────┘
   Git-backed   Issue     LSP       Language
   messaging   tracking  navigation
```

## Project Structure

```
open-swarm/
├── opencode.json          # OpenCode configuration
├── AGENTS.md             # Agent instructions
├── .opencode/
│   ├── tool/            # Custom tools (Beads integration)
│   ├── command/         # Slash commands
│   ├── agent/           # Custom agent definitions
│   └── plugin/          # OpenCode plugins
├── .beads/
│   └── issues.jsonl     # Beads issue tracking (Git-committed)
├── cmd/
│   └── open-swarm/      # CLI application
├── pkg/
│   ├── coordinator/     # Coordination logic
│   ├── agent/          # Agent management
│   └── tasks/          # Task management
├── internal/
│   └── config/         # Configuration
└── docs/               # Documentation
```

## Development

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building

```bash
# Development
go build -o bin/open-swarm ./cmd/open-swarm

# Production (optimized)
go build -ldflags="-s -w" -o bin/open-swarm ./cmd/open-swarm
```

### Code Quality

```bash
# Format
gofmt -w .

# Lint
golangci-lint run

# All checks
gofmt -w . && golangci-lint run && go test ./...
```

## Multi-Agent Workflows

### Parallel Development

Multiple agents work on independent features simultaneously:

```
Agent A: Backend API (reserves internal/api/**)
Agent B: Frontend (reserves web/**)
Agent C: Tests (reserves test/**)

Each agent:
1. /session-start
2. /task-start <their-task-id>
3. Work independently
4. /coordinate <other-agent> when ready for integration
5. /session-end
```

### Sequential Pipeline

Work flows through stages handled by different agents:

```
Agent A: Schema Design → /coordinate Agent B "Schema ready"
    ↓
Agent B: Migrations → /coordinate Agent C "Migrations ready"
    ↓
Agent C: Testing → Complete task
```

### Code Review

```
Agent A: Implements feature → /coordinate reviewer "Review needed"
    ↓
Reviewer Agent: Reviews code → Provides feedback
    ↓
Agent A: Addresses feedback → /coordinate reviewer "Changes applied"
    ↓
Reviewer Agent: Approves → Agent A closes task
```

## Best Practices

### File Reservations

- Always reserve files before editing
- Use specific patterns, not broad globs
- Release promptly when done
- Renew if work takes longer than 1 hour

### Task Management

- Break work into small, focused tasks
- Update Beads status frequently
- File discovered issues immediately
- Link tasks with dependencies

### Communication

- Use clear, specific subjects
- Include task IDs in subjects: `[bd-a1b2] Feature complete`
- Set appropriate importance levels
- Require acks for critical coordination

### Session Hygiene

- Always run `/session-start` at beginning
- Always run `/session-end` when finished
- Never leave file reservations held
- Always sync Beads to Git

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

### OpenCode Not Finding MCP Servers

```bash
# Verify MCP servers can start
python -m mcp_agent_mail.server &
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --cwd . &

# Check opencode.json MCP configuration
cat opencode.json | grep -A 10 '"mcp"'
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

## Resources

- [OpenCode Documentation](https://opencode.ai/docs/)
- [Agent Mail GitHub](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [Beads GitHub](https://github.com/steveyegge/beads)
- [Serena Documentation](https://oraios.github.io/serena/)
- [AGENTS.md Format](https://agents.md)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

## License

[Your License Here]

## Contributing

1. Check `/task-ready` for available work
2. `/task-start <task-id>`
3. `/reserve` files you'll edit
4. Make changes following AGENTS.md guidelines
5. Run tests: `go test ./...`
6. `/task-complete <task-id>`
7. `/session-end`

## Support

For issues:
- Check AGENTS.md for detailed workflows
- Review opencode.json configuration
- Check MCP server status
- File issue in Beads: `bd create "Issue description" -t bug`
