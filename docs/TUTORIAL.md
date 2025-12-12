# Open Swarm Tutorial: Your First Multi-Agent Workflow

Welcome! This tutorial will walk you through everything you need to get started with Open Swarm in about 30 minutes. By the end, you'll have run your first workflow and understand how multi-agent coordination works.

## Table of Contents

1. [What is Open Swarm?](#what-is-open-swarm)
2. [Installing Prerequisites](#installing-prerequisites)
3. [Setting Up Your Project](#setting-up-your-project)
4. [Your First Session](#your-first-session)
5. [Understanding Results](#understanding-results)
6. [Modifying Workflows](#modifying-workflows)
7. [Debugging Failures](#debugging-failures)
8. [Next Steps](#next-steps)

---

## What is Open Swarm?

Open Swarm is a framework that lets multiple AI coding agents work together on the same codebase **without conflicts**. Think of it like this:

- **Traditional development:** One person edits files, conflicts happen if two people edit the same file
- **Open Swarm:** Multiple AI agents work simultaneously on different files, coordinating through messages and task tracking

Open Swarm handles the hard parts:
- File locking (so agents don't overwrite each other)
- Task scheduling (agents know what to work on)
- Inter-agent communication (agents send messages)
- Progress tracking (see what's done)

### Key Components

- **OpenCode** - The AI agent platform (runs the agents)
- **Agent Mail** - Git-backed messaging system (agents send messages to each other)
- **Beads** - Task/issue tracking system (tracks what needs doing)
- **Serena** - Code understanding tool (agents understand code better)

---

## Installing Prerequisites

Install these tools in order. Each step has a verification command to confirm it worked.

### 1. Install Go 1.25+

Go is the programming language this project uses.

**macOS (using Homebrew):**
```bash
brew install go
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install golang-go
```

**Windows:** Download from https://go.dev/dl/

**Verify:**
```bash
go version
# Should output: go version go1.25.x ...
```

### 2. Install OpenCode (SST)

OpenCode is the AI agent platform - it runs the agents that do the work.

```bash
curl -fsSL https://opencode.ai/install | bash
```

**Verify:**
```bash
opencode --version
# Should show a version like: opencode version X.Y.Z
```

### 3. Install Agent Mail

Agent Mail is the messaging system that lets agents communicate and coordinate.

```bash
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh?$(date +%s)" | bash -s -- --yes
```

This creates an `am` command for you.

**Verify:**
```bash
# In a separate terminal, start Agent Mail
am

# Should output:
# Agent Mail MCP server running on http://localhost:8765

# Stop it with Ctrl+C for now
```

### 4. Install Beads

Beads is the task tracking system - it tracks what work needs to be done.

```bash
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
```

**Verify:**
```bash
bd --version
# Should output a version number
```

### 5. Install Serena

Serena helps agents understand code structure. It needs Python's `uv` package manager.

```bash
# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Test Serena (will download on first run - this is normal!)
uvx --from git+https://github.com/oraios/serena serena --help
```

**Verify:**
```bash
which uv
# Should show the path to uv

uvx --from git+https://github.com/oraios/serena serena --help
# Should show Serena help text
```

### 6. Verify All Prerequisites

Run this command to check everything is installed:

```bash
echo "=== Go ===" && \
go version && \
echo "=== OpenCode ===" && \
opencode --version && \
echo "=== Beads ===" && \
bd --version && \
echo "=== Python/uv ===" && \
uv --version && \
echo "✅ All prerequisites installed!"
```

**Did something fail?** Jump to the [Debugging Failures](#debugging-failures) section.

---

## Setting Up Your Project

Now that you have the tools, let's set up the Open Swarm project itself.

### 1. Navigate to the Project

```bash
cd /home/lewis/src/open-swarm
```

### 2. Install Go Dependencies

This downloads the Go libraries the project needs.

```bash
go mod download
```

**What this does:** Reads `go.mod` and downloads all required packages (like Temporal, task libraries, etc.)

### 3. Initialize Beads

Beads tracks tasks in the project.

```bash
bd init
```

**What this does:** Creates `.beads/issues.jsonl` (Git-tracked file that stores tasks)

### 4. Build the CLI

Compile the Open Swarm command-line tool.

```bash
go build -o bin/open-swarm ./cmd/open-swarm
```

**What this does:** Reads the Go code in `cmd/open-swarm` and creates an executable `bin/open-swarm`

### 5. Run Tests (Optional but Recommended)

Make sure everything compiles and works.

```bash
go test ./...
```

**Expected output:** Should see something like:
```
ok  	open-swarm/cmd/open-swarm	0.123s
ok	open-swarm/pkg/coordinator	0.456s
...
```

**✅ Setup Complete!** You're ready to run your first session.

---

## Your First Session

This is the moment of truth! You'll start two processes (Agent Mail and OpenCode), then run your first workflow.

### Step 1: Start Agent Mail (Background Server)

Agent Mail needs to be running for agents to communicate.

**In Terminal 1:**
```bash
cd /home/lewis/src/open-swarm
am
```

**You should see:**
```
Agent Mail MCP server running on http://localhost:8765
```

**Leave this running.** Don't close this terminal - Agent Mail needs to stay active.

### Step 2: Start OpenCode (Agent Platform)

**In Terminal 2** (in the same project directory):
```bash
cd /home/lewis/src/open-swarm
opencode
```

**You should see:**
```
OpenCode Terminal UI
...
```

You're now in the OpenCode interactive interface (a terminal UI).

### Step 3: Configure Your AI Provider (First Time Only)

First time you run OpenCode, it needs to know which AI model to use.

**Type:**
```
/connect
```

You'll see options:
- **OpenCode Zen** (easiest - curated models)
- Anthropic Claude (if you have API key)
- OpenAI (if you have API key)
- Google Gemini (if you have API key)

**Pick one** and follow the prompts to enter your API key. (If you don't have an API key, sign up at the provider's website - usually takes 5 minutes.)

### Step 4: Start Your Session

This initializes everything - registers your agent, checks messages, loads tasks, etc.

**Type:**
```
/session-start
```

**You'll see output like:**
```
🤖 Agent Identity: BlueLake
   (This is your agent's unique name)

📬 Inbox: 0 messages
   (No messages from other agents yet)

📋 Ready Tasks: 3 available
   (Work available in Beads)

👥 Active Agents: 1
   (Just you right now)

📁 File Reservations: None
   (No files locked)

✨ Next Steps:
   1. Check ready tasks with /task-ready
   2. Start a task with /task-start [task-id]
   3. Read/edit files
   4. End session with /session-end
```

**Great!** Your session is initialized. Your agent (e.g., "BlueLake") is now registered and ready to work.

### Step 5: Check Available Work

See what tasks are available.

**Type:**
```
/task-ready
```

**You'll see something like:**
```
Available Tasks:
  bd-a1b2: Add authentication handler (priority: high)
  bd-c3d4: Write unit tests (priority: medium)
  bd-e5f6: Fix null pointer bug (priority: high)
```

### Step 6: Start a Task

Let's work on a task. Pick one from the list and start it.

**Type:**
```
/task-start bd-a1b2
```

**What happens:**
1. Task status changes to `in_progress` in Beads
2. Related files are automatically reserved (locked for your use)
3. You see the task description

**Output:**
```
Task bd-a1b2: Add authentication handler
Status: in_progress
Files reserved: internal/api/**/*.go
```

### Step 7: Make a Change

Now let's actually work on the code. You can:

**Find a function:**
```
Find symbol: "UserService"
```

**Read a file:**
```
Read file: internal/api/handlers.go
```

**Edit a file:**
```
Edit internal/api/handlers.go to add error handling to login endpoint
```

**Run tests:**
```
!go test ./internal/api/...
```

(The `!` prefix runs shell commands)

OpenCode will:
1. Find the code
2. Show you the current version
3. Make the changes
4. Run tests
5. Show results

### Step 8: Complete Your Task

When you're done with the task:

**Type:**
```
/task-complete bd-a1b2
```

**What happens:**
1. Task status changes to `completed` in Beads
2. Files are automatically released (no longer locked)
3. You get a summary

### Step 9: End Your Session

Always end properly - this cleans up and syncs everything.

**Type:**
```
/session-end
```

**What happens:**
1. Any remaining file reservations are released
2. Beads is synced to Git
3. You get a handoff summary showing what was accomplished
4. Messages about the session are stored for other agents

**You're done!** Your session is complete.

---

## Understanding Results

Now let's understand what happened and where to find the results.

### Session Output Explained

When you run `/session-start`, you see:

```
🤖 Agent Identity: BlueLake
```
- This is your agent's unique name (auto-generated, memorable)
- Used in all coordination with other agents

```
📬 Inbox: 2 messages (1 urgent)
```
- Messages from other agents
- "Urgent" means they're high-priority
- Check and respond to messages as needed

```
📋 Ready Tasks: 5 available
```
- Tasks that have no dependencies and can start immediately
- Use `/task-ready` to see the full list
- Not shown: blocked tasks (waiting for dependencies)

```
👥 Active Agents: 3
   - BlueLake (you, active right now)
   - GreenForest (working on migrations)
   - RedMountain (inactive for 2 hours)
```
- Other agents in the system
- Shows their status and when last active

```
📁 File Reservations:
   - internal/api/** (GreenForest, expires in 45min)
   - web/src/** (available)
```
- Files currently locked by agents
- Shows who has them and when they expire
- Reserve before editing to avoid conflicts

### Task Results in Beads

After `/task-complete`, check the task status:

```bash
bd list --json | grep -A 10 "bd-a1b2"
```

**Output shows:**
```json
{
  "id": "bd-a1b2",
  "title": "Add authentication handler",
  "status": "completed",
  "completed_at": "2025-12-12T14:30:00Z",
  "summary": "Added JWT validation to login endpoint"
}
```

### File Reservations

Check who has which files reserved:

```bash
# In OpenCode terminal
# Files are shown in /session-start output

# Or query Agent Mail directly:
curl http://localhost:8765/file-reservations
```

### Agent Mail Messages

Agents communicate via messages. Check your inbox:

```bash
# In OpenCode terminal
# Messages appear in /session-start output
# /coordinate sends messages to other agents
```

Messages are stored in `~/.agent-mail/archive/messages/` in Git format.

### Git-Backed Artifacts

Everything is committed to Git for audit trail:

```bash
# View recent commits
git log --oneline | head -20

# See what changed
git diff HEAD~5..HEAD
```

You'll see commits like:
```
3a2b1c Message: BlueLight -> GreenForest "API ready for integration"
2f1e0d Task bd-a1b2 completed
1d0c9b File reservation: internal/api/**
```

---

## Modifying Workflows

Now you understand the basics. Let's explore customizing workflows.

### 1. Creating New Tasks

Instead of working on existing tasks, create your own.

**In OpenCode:**
```
/coordinate Team "Need to add caching layer"
```

**Or using Beads directly:**
```bash
cd /home/lewis/src/open-swarm

# Create a new task
bd create "Implement Redis caching" -t feature

# Create with priority
bd create "Fix database deadlock" -t bug -p high

# Create with dependencies
bd create "Write tests for caching" -t test -d bd-f1g2
```

### 2. Creating Tasks with Dependencies

Tasks can depend on each other. If Task B depends on Task A, Task B won't show in `/task-ready` until A is done.

```bash
# Task A: Schema migration
bd create "Create users table" -t feature --id bd-a1b2

# Task B: Depends on A
bd create "Add user authentication" -t feature -d bd-a1b2

# Now bd-a1b2 must complete before bd-xxxx shows as ready
```

**In OpenCode:**
```
/task-ready
# Won't show "Add user authentication" yet

/task-complete bd-a1b2
# Now it appears in /task-ready
```

### 3. Reserving Specific Files

By default, `/task-start` reserves related files. But you can manually control reservations.

**Reserve files before editing:**
```
/reserve internal/api/handlers.go
```

**Reserve with pattern:**
```
/reserve pkg/auth/**/*.go
```

**Reserve multiple patterns:**
```
/reserve internal/api/**, pkg/database/**
```

**Release all:**
```
/release
```

**Release specific:**
```
# Currently not in OpenCode, but using Beads:
# File reservations are released when tasks complete
```

### 4. Multi-Agent Workflows

The real power: multiple agents working in parallel!

#### Scenario: Backend + Frontend

**Terminal 1 - Agent (Backend):**
```bash
opencode
/session-start
# Gets BlueLake identity

/task-ready
# Shows: "Build REST API"

/task-start bd-api

/reserve internal/api/**, pkg/database/**
# Edit files
# Work on API...

# When API is ready:
/coordinate GreenForest "REST API ready for integration"

/task-complete bd-api
/session-end
```

**Terminal 2 - Agent (Frontend):**
```bash
opencode
/session-start
# Gets GreenForest identity

# Check inbox - see message from BlueLake
# Inbox shows: BlueLake: "REST API ready for integration"

/task-ready
# Shows: "Build web UI"

/task-start bd-ui

/reserve web/**, assets/**
# Edit files
# Build UI using the API that BlueLake created

/task-complete bd-ui
/session-end
```

**Key points:**
- Agents work on different files (no conflicts)
- Agent Mail messages coordinate handoffs
- Beads tracks dependencies
- File reservations prevent accidents

#### Scenario: Sequential Pipeline

Work flows through stages (common in CI/CD):

**Stage 1: Design (Agent A)**
```
/task-start bd-schema
# Design database schema
/coordinate AgentB "Schema ready at pkg/models/schema.go"
/task-complete bd-schema
```

**Stage 2: Migrations (Agent B)**
```
# Wait for Agent A's message
/task-start bd-migration
# Write migrations using the schema
/coordinate AgentC "Migrations ready, database is set up"
/task-complete bd-migration
```

**Stage 3: Testing (Agent C)**
```
# Wait for Agent B's message
/task-start bd-integration-tests
# Write tests with real database
/task-complete bd-integration-tests
```

### 5. Using Custom Agents

Open Swarm comes with specialized agents. Invoke them with `@`:

```
@reviewer - Code review specialist
  /review internal/api/handlers.go
  @reviewer please review the JWT validation changes

@coordinator - Coordination specialist
  @coordinator help schedule work between teams

@tester - Test writing specialist
  @tester write tests for the authentication module
```

---

## Debugging Failures

Things don't always work perfectly. Here's how to diagnose and fix common issues.

### Failure: "Agent Mail not connected"

**Symptom:**
```
Error: Cannot connect to Agent Mail
```

**Why:**
- Agent Mail server isn't running
- Server crashed
- Wrong port

**Fix:**
```bash
# Check if running
curl http://localhost:8765/health

# If error, start it
am

# Wait for startup
# Should see: Agent Mail MCP server running on http://localhost:8765

# Then try OpenCode again
```

### Failure: "OpenCode can't find MCP servers"

**Symptom:**
```
Error: Cannot connect to MCP servers
MCP tool not found: beads.ready
```

**Why:**
- Agent Mail isn't running
- MCP server configuration is wrong in `opencode.json`
- Serena/uv not installed

**Fix:**
```bash
# 1. Check Agent Mail is running
curl http://localhost:8765/health

# 2. Validate opencode.json
cat opencode.json | python -m json.tool
# Should parse without errors

# 3. Test Serena
uvx --from git+https://github.com/oraios/serena serena --help

# 4. Restart OpenCode
```

### Failure: "File reservation conflict"

**Symptom:**
```
Cannot reserve internal/api/handlers.go: held by GreenForest (45 min remaining)
```

**Why:**
- Another agent is editing the same file
- Their reservation hasn't expired yet

**Options:**
```bash
# Option 1: Wait (reservations expire after 1 hour)
sleep 45  # Wait 45 minutes, then try again

# Option 2: Coordinate with the other agent
/coordinate GreenForest "Need to work on handlers.go, when will you be done?"

# Option 3: Work on different files meanwhile
/task-ready  # Get other tasks

# Option 4: Urgent - message asking them to release
/coordinate GreenForest "Critical bug fix needed in handlers.go, can you release?"
```

### Failure: "Beads command not found"

**Symptom:**
```
Command not found: bd
```

**Why:**
- Beads not installed or PATH is wrong

**Fix:**
```bash
# Reinstall Beads
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Verify
bd --version

# If still failing, add to PATH:
export PATH="$HOME/.local/bin:$PATH"
bd --version
```

### Failure: "No tasks available"

**Symptom:**
```
/task-ready
No ready tasks available
```

**Why:**
- All tasks are complete
- All remaining tasks are blocked by dependencies
- Tasks haven't been created yet

**Check:**
```bash
# See all tasks
bd list

# See details
bd list --json | python -m json.tool

# Filter by status
bd list --json | grep -i "status" | head -20
```

**Fix:**
```bash
# Create new tasks
bd create "New feature: user profiles" -t feature

# Unblock dependencies
/task-complete bd-blocking-task

# Check again
/task-ready
```

### Failure: "Go test failed"

**Symptom:**
```
!go test ./...
FAIL: package test failed
```

**Why:**
- Code has bugs (OpenCode can help fix!)
- Tests are outdated
- Dependencies missing

**Debug:**
```bash
# Run tests with details
!go test -v ./...

# Run specific test
!go test -v -run TestNamePattern ./...

# Check for build errors
!go build ./...

# See what OpenCode suggests
# (OpenCode analyzes test failures and can fix them)
```

### Failure: "Temporal/Docker Issues"

If you're using Temporal for advanced workflows:

```bash
# Check Docker services
docker-compose ps

# See logs
docker-compose logs -f temporal

# Restart
docker-compose down
docker-compose up -d
sleep 30

# Verify health
docker-compose exec temporal tctl --address temporal:7233 cluster health
```

### General Debugging Steps

1. **Gather information:**
   ```bash
   # System status
   echo "=== System ===" && \
   docker-compose ps && \
   echo "=== Agent Mail ===" && \
   curl http://localhost:8765/health && \
   echo "=== Beads ===" && \
   bd list --json | head -5
   ```

2. **Check logs:**
   ```bash
   # Agent Mail logs
   tail -50 ~/.agent-mail/server.log

   # Docker logs
   docker-compose logs -f --tail=50

   # Go build log
   go build -v ./... 2>&1 | tee build.log
   ```

3. **Verify prerequisites:**
   ```bash
   # Run the verification script
   go version && opencode --version && bd --version && uv --version
   ```

4. **Check configuration:**
   ```bash
   # Validate opencode.json
   cat opencode.json | python -m json.tool

   # Check .beads/issues.jsonl exists
   ls -la .beads/issues.jsonl

   # Check git is initialized
   git status
   ```

5. **Try a clean restart:**
   ```bash
   # In separate terminals:

   # Terminal 1 (stop everything)
   pkill -f "am"
   pkill -f "opencode"

   # Terminal 2 (start Agent Mail)
   am

   # Terminal 3 (start OpenCode)
   opencode
   ```

---

## Next Steps

Congratulations! You've completed the tutorial. Here's what to learn next:

### 1. Read AGENTS.md

This document has comprehensive information about:
- Architecture and design patterns
- Development workflows for your specific project
- Code standards and testing
- Multi-agent coordination best practices

**Read it:** `cat AGENTS.md | less`

### 2. Explore the Commands

Open Swarm has many more commands. In OpenCode, type:

```
/help
```

Or in `.opencode/command/` you'll find:
- `session-start.md` - Session initialization protocol
- `session-end.md` - Session cleanup protocol
- `coordinate.md` - Inter-agent messaging

### 3. Try Multi-Agent Workflows

1. Open two terminals
2. In Terminal 1: `opencode` → `/session-start` → work on backend
3. In Terminal 2: `opencode` → `/session-start` → work on frontend
4. Use `/coordinate` to send messages between them

### 4. Create Custom Tasks

```bash
# Create tasks for your team to work on
bd create "Implement caching layer" -t feature -p high
bd create "Add rate limiting" -t feature
bd create "Write integration tests" -t test
```

### 5. Check Out Examples

Look at example workflows:

```bash
ls examples/
cat examples/README.md
cat docs/DAG-WORKFLOW.md    # Complex parallel workflows
cat docs/TCR-WORKFLOW.md    # Test-Commit-Revert workflow
```

### 6. Review Advanced Topics

- `docs/DEPLOYMENT.md` - Running agents in production
- `docs/MONITORING.md` - Tracking agent activity
- `docs/TROUBLESHOOTING.md` - Deep debugging guide

### 7. Join Multi-Agent Development

Once you're comfortable:
1. Check what tasks are available: `/task-ready`
2. Start working: `/task-start bd-xxxx`
3. Coordinate with others: `/coordinate AgentName "message"`
4. Ship your work: `/task-complete bd-xxxx`

---

## Quick Reference

### Session Lifecycle

```bash
# Every session starts with:
/session-start

# Work on tasks:
/task-ready             # See available work
/task-start bd-xxxx     # Begin task
/reserve patterns/**    # Lock files
!command               # Run shell
/task-complete bd-xxxx # Finish task

# Every session ends with:
/session-end
```

### File Operations

```bash
# Reserve files before editing
/reserve pkg/auth/**/*.go

# Release when done
/release

# Check who has what reserved
# (Shown in /session-start output)
```

### Coordination

```bash
# Send message to another agent
/coordinate BlueLake "API implementation complete"

# Check inbox
# (Shown in /session-start output)

# Use custom agents
@reviewer        # Code review
@tester         # Test writing
@coordinator    # Coordination help
```

### Beads (Task Tracking)

```bash
# Check what work is available
/task-ready

# Create new task
bd create "Description" -t feature

# See all tasks
bd list

# See task details
bd list --json

# Update task status
bd status bd-xxxx in_progress
```

---

## Troubleshooting Quick Links

- **Agent Mail won't start:** See [Failure: Agent Mail not connected](#failure-agent-mail-not-connected)
- **Files locked by other agent:** See [Failure: File reservation conflict](#failure-file-reservation-conflict)
- **Beads command not found:** See [Failure: Beads command not found](#failure-beads-command-not-found)
- **Tests failing:** See [Failure: Go test failed](#failure-go-test-failed)
- **More issues:** See `docs/TROUBLESHOOTING.md` for comprehensive guide

---

## Getting Help

If you're stuck:

1. **Read the relevant doc:**
   - `README.md` - Overview
   - `QUICKSTART.md` - 10-minute setup
   - `AGENTS.md` - Project-specific info
   - `docs/TROUBLESHOOTING.md` - Deep troubleshooting

2. **Check logs:**
   ```bash
   tail -50 ~/.agent-mail/server.log
   docker-compose logs -f
   ```

3. **File a Beads issue:**
   ```bash
   bd create "Problem description" -t bug
   ```

4. **Message for help:**
   ```
   /coordinate @coordinator "Help with [issue]"
   ```

---

## Summary

You now know:

✅ How to install all prerequisites
✅ How to set up the Open Swarm project
✅ How to run your first session
✅ How to understand results and outputs
✅ How to modify workflows and create tasks
✅ How to debug common failures
✅ Where to find more information

**You're ready to start building with Open Swarm!**

Start with a simple task, get comfortable with the workflow, then try coordinating with other agents. The framework handles all the coordination complexity - you focus on writing great code.

Happy building! 🚀
