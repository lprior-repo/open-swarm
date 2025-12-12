# Open Swarm Quick Start

Get up and running with multi-agent coordination in 10 minutes.

## Prerequisites

Install the required tools in this order:

### 1. Go 1.25+

```bash
# Verify installation
go version
# Should show: go version go1.25.x ...
```

### 2. OpenCode (SST)

```bash
# Install OpenCode
curl -fsSL https://opencode.ai/install | bash

# Verify
opencode --version
```

### 3. Agent Mail MCP Server

```bash
# Install (creates 'am' alias)
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh?$(date +%s)" | bash -s -- --yes

# Test (should start server)
am
# Press Ctrl+C to stop, or leave running in separate terminal
```

### 4. Beads

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Verify
bd --version
```

### 5. Serena

```bash
# Install uv package manager
curl -LsSf https://astral.sh/uv/install.sh | sh

# Test Serena (will download on first run)
uvx --from git+https://github.com/oraios/serena serena --help
```

## Project Setup

```bash
# Navigate to project
cd /home/lewis/src/open-swarm

# Install Go dependencies
go mod download

# Initialize Beads (if not already done)
bd init

# Build the CLI
go build -o bin/open-swarm ./cmd/open-swarm

# Verify tests pass
go test ./...
```

## Your First Session

### Step 1: Start Agent Mail

In a separate terminal:

```bash
am
```

Keep this running. You should see:
```
Agent Mail MCP server running on http://localhost:8765
```

### Step 2: Start OpenCode

In your project directory:

```bash
opencode
```

You'll see the OpenCode TUI (terminal user interface).

### Step 3: Configure Provider

First time only:

```
/connect
```

Select a provider:
- **OpenCode Zen** (easiest - curated models)
- Anthropic Claude
- OpenAI
- Google Gemini

Enter your API key when prompted.

### Step 4: Initialize Session

```
/session-start
```

This command will:
1. ✅ Register your agent with Agent Mail
2. ✅ Fetch your inbox
3. ✅ Check for ready tasks in Beads
4. ✅ List active agents
5. ✅ Show file reservations
6. ✅ Provide next steps

You'll receive a summary like:

```
🤖 Agent Identity: BlueLake
📬 Inbox: 0 messages
📋 Ready Tasks: 3 available
👥 Active Agents: 1 (just you!)
📁 File Reservations: None
✨ Suggested: Start task bd-a1b2 (highest priority)
```

### Step 5: Start Working on a Task

```
/task-ready
```

Shows unblocked tasks from Beads. Pick one and start it:

```
/task-start bd-a1b2
```

This will:
- Update Beads task status to `in_progress`
- Reserve related files automatically
- Show task details

### Step 6: Make Changes

Now work normally! OpenCode helps you:

```
# Find code
Find symbol: "UserService"

# Read files
Read the file: internal/api/handlers.go

# Edit code
Edit internal/api/handlers.go to add validation

# Run tests
!go test ./internal/api/...
```

**Serena** (if working) provides semantic navigation:
- Symbol finding
- Reference lookup
- LSP-powered code understanding

### Step 7: Complete Your Work

When done:

```
/task-complete bd-a1b2
```

This will:
- Close the task in Beads
- Release file reservations
- Ask for completion reason

Then end your session:

```
/session-end
```

This performs cleanup:
- Files discovered issues in Beads
- Releases any remaining file reservations
- Syncs Beads to Git
- Provides handoff summary

## Working with Multiple Agents

### Scenario: Backend + Frontend Development

**Terminal 1 - Agent A (Backend):**

```bash
opencode
/session-start
/task-start bd-backend-api
# Work on backend...
/coordinate FrontendAgent "API endpoints ready"
/session-end
```

**Terminal 2 - Agent B (Frontend):**

```bash
opencode
/session-start
# See message from Agent A in inbox
/task-start bd-frontend-ui
# Reserve different files
/reserve web/src/**/*.ts
# Work on frontend...
/session-end
```

## Essential Commands Reference

### Session Management

| Command | When to Use |
|---------|-------------|
| `/session-start` | Beginning of every session |
| `/session-end` | End of every session |
| `/sync` | Manually sync with Agent Mail |

### Task Management

| Command | When to Use |
|---------|-------------|
| `/task-ready` | Check for available work |
| `/task-start <id>` | Begin working on a task |
| `/task-complete <id>` | Finish a task |

### File Coordination

| Command | When to Use |
|---------|-------------|
| `/reserve <pattern>` | Before editing files |
| `/release` | When done editing |

### Communication

| Command | When to Use |
|---------|-------------|
| `/coordinate <agent> <subject>` | Send message to another agent |
| Check inbox | See incoming messages |

### Code Operations

| Command | When to Use |
|---------|-------------|
| `/review <files>` | Request code review |
| `@reviewer` | Invoke review agent |
| `@coordinator` | Get coordination help |
| `@tester` | Get test writing help |

## Understanding the Output

### Session Start Summary

```
🤖 Agent Identity: BlueLake
   - Your unique agent name (auto-generated)
   - Used for all coordination

📬 Inbox: 2 messages (1 urgent)
   - Messages from other agents
   - Check and acknowledge urgent ones

📋 Ready Tasks: 5 available
   - Unblocked work from Beads
   - Prioritized by importance

👥 Active Agents: 3
   - BlueLake (you)
   - GreenForest (working on migrations)
   - RedMountain (inactive, 2 hours ago)

📁 File Reservations:
   - internal/api/** (GreenForest, expires in 45min)
   - web/src/** (available)

✨ Next Steps:
   1. Review urgent message from GreenForest
   2. Start highest priority task: bd-a1b2
   3. Reserve files before editing
```

## Common Workflows

### Solo Development

```bash
opencode
/session-start
/task-ready                    # Check available work
/task-start bd-a1b2           # Start task
/reserve pkg/**/*.go          # Reserve files
# ... make changes ...
!go test ./...                # Run tests
/task-complete bd-a1b2        # Finish
/session-end
```

### Code Review Request

```bash
# After implementing feature
/reserve internal/api/**/*.go
# ... implement ...
/coordinate reviewer "Review authentication changes"
# Reviewer agent will analyze and respond
/session-end
```

### Debugging with Another Agent

```bash
/coordinate DebugAgent "Help with null pointer in user handler"
# Work together via messages
# DebugAgent might reserve files and investigate
/session-end
```

## Tips for Success

### Always Reserve Files

```bash
# Good - specific pattern
/reserve internal/api/handlers.go

# Good - directory pattern
/reserve pkg/auth/**/*.go

# Bad - too broad
# /reserve **/*.go  (conflicts with everyone!)
```

### Keep Tasks Small

```bash
# Good
bd create "Add JWT validation" -t feature
bd create "Add error handling to login" -t bug

# Bad
bd create "Implement entire authentication system"
```

### Communicate Proactively

```bash
# When you complete blocking work
/coordinate FrontendAgent "API ready for integration"

# When you need help
/coordinate TeamLead "Stuck on database migration"

# When you're done for the day
/coordinate Team "Completed bd-a1b2, bd-a1b3 ready for next session"
```

### Check Your Inbox

```bash
# At start of session
/session-start  (checks automatically)

# Periodically during work
Check Agent Mail inbox

# Before ending
/session-end  (sends status update)
```

## Troubleshooting

### "Agent Mail not connected"

```bash
# In separate terminal, start Agent Mail
am

# Should see:
# Agent Mail MCP server running on http://localhost:8765
```

### "OpenCode can't find MCP servers"

```bash
# Check opencode.json exists
cat opencode.json

# Verify MCP server commands work
python -m mcp_agent_mail.server &
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --cwd . &
```

### "Beads command not found"

```bash
# Re-install Beads
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Initialize in project
bd init
```

### "File reservation conflict"

```
Check who has the reservation via Agent Mail

Options:
1. Wait for TTL (1 hour default)
2. Message the holding agent
3. Work on different files
```

## Next Steps

Now that you're up and running:

1. **Read AGENTS.md** - Comprehensive guide for this project
2. **Explore custom commands** - See all available slash commands
3. **Try multi-agent workflows** - Start multiple OpenCode sessions
4. **Check out the tools** - `.opencode/tool/beads.ts` for Beads integration

## Quick Command Cheatsheet

```bash
# Every session
/session-start          # Always first
/session-end           # Always last

# Tasks
/task-ready            # What can I work on?
/task-start <id>       # Start work
/task-complete <id>    # Finish work

# Files
/reserve <pattern>     # Lock files
/release               # Unlock files

# Coordination
/coordinate <agent> <subject>   # Send message
Check inbox                     # Read messages

# Code
/review <files>        # Request review
@reviewer             # Invoke reviewer agent
!command              # Run shell command
```

That's it! You're ready to start coordinating with other agents. Happy coding! 🚀
