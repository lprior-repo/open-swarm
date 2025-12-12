# Open Swarm Troubleshooting Guide

This guide covers common issues encountered when working with Open Swarm and provides solutions and debugging commands.

## Table of Contents

1. [Port Conflicts](#port-conflicts)
2. [Temporal Connection Issues](#temporal-connection-issues)
3. [OpenCode Server Failures](#opencode-server-failures)
4. [Worktree Cleanup](#worktree-cleanup)
5. [Common Errors](#common-errors)
6. [Debugging Commands](#debugging-commands)

---

## Port Conflicts

### Problem: Port Already in Use

When starting Temporal or other services, you may encounter an error like:

```
Error: listen EADDRINUSE :::7233
```

Or for Agent Mail:
```
Address already in use
```

### Causes

- Previous service instance still running
- Another application using the same port
- Port not properly released after shutdown

### Solutions

#### Check and Kill Running Processes

```bash
# Find process using Temporal port (7233)
lsof -i :7233

# Find process using Agent Mail port (8765)
lsof -i :8765

# Find process using PostgreSQL port (5432)
lsof -i :5432

# Find process using Temporal web UI port (8233)
lsof -i :8233

# Kill a process (replace PID with the actual process ID)
kill -9 <PID>

# Or kill by name
pkill -9 -f "temporal"
pkill -9 "java"  # Temporal runs on Java
```

#### Start Services in Clean State

```bash
# Stop all Docker containers
docker-compose down

# Remove any dangling containers
docker container prune -f

# Start fresh
docker-compose up -d
```

#### Use Different Ports

If you need to run multiple instances, configure different ports in `docker-compose.yml`:

```yaml
services:
  temporal:
    ports:
      - "7234:7233"  # Map to different port
      - "8234:8233"
```

Then update your connection configuration to use the new ports.

#### Check Port Availability

```bash
# List all listening ports
netstat -tuln | grep LISTEN

# On macOS
lsof -i -P -n | grep LISTEN

# Check specific port status
ss -tlnp | grep 7233
```

### Prevention

- Always use `docker-compose down` instead of `Ctrl+C` to properly shutdown services
- Don't run multiple instances of Temporal without changing ports
- Check port availability before starting services

---

## Temporal Connection Issues

### Problem: Cannot Connect to Temporal Server

Error messages:
```
Unable to connect to Temporal server
Connection refused
deadline exceeded
```

### Causes

- Temporal server not running or not fully initialized
- Network connectivity issues
- Database (PostgreSQL) not ready
- Incorrect connection configuration

### Solutions

#### Verify Temporal is Running

```bash
# Check container status
docker-compose ps

# Expected output should show temporal and postgresql as UP

# Check Temporal health
docker-compose logs temporal

# Look for messages indicating successful startup
```

#### Wait for Full Initialization

Temporal requires PostgreSQL to be ready first. This can take 30+ seconds.

```bash
# Wait for services to be healthy
docker-compose up -d && sleep 30

# Check health status
docker-compose exec temporal tctl --address temporal:7233 cluster health
```

#### Verify Database Connection

```bash
# Check if PostgreSQL is running
docker-compose ps postgresql

# Connect to PostgreSQL directly
docker-compose exec postgresql psql -U temporal -d postgres -c "SELECT 1"

# Check PostgreSQL logs
docker-compose logs postgresql
```

#### Test Connection from Your Application

```bash
# From within the Docker network
docker-compose exec temporal tctl --address temporal:7233 cluster health

# Or from your Go application:
# Update connection string in your code to:
# "temporal:7233" (for Docker)
# "localhost:7233" (for local connections)
```

#### Rebuild and Restart

```bash
# Clean everything
docker-compose down -v

# Rebuild
docker-compose build --no-cache

# Start fresh
docker-compose up -d

# Wait for health checks to pass
sleep 40

# Verify
docker-compose exec temporal tctl --address temporal:7233 cluster health
```

#### Check Network Configuration

```bash
# Verify services can communicate
docker-compose exec temporal ping postgresql

# Check Temporal configuration
docker-compose logs temporal | grep -i "config\|error\|connection"
```

### Debugging Commands

```bash
# Get detailed Temporal server logs
docker-compose logs temporal -f --tail=100

# Get PostgreSQL logs
docker-compose logs postgresql -f --tail=50

# List all Docker networks
docker network ls

# Inspect the network Open Swarm uses
docker network inspect <network-name>

# Check environment variables in containers
docker-compose exec temporal env | grep -i temporal
docker-compose exec postgresql env | grep -i postgres
```

### Prevention

- Always start with `docker-compose up -d` and wait at least 30 seconds
- Check `docker-compose ps` before connecting
- Review `AGENTS.md` for proper configuration
- Keep Docker and docker-compose updated

---

## OpenCode Server Failures

### Problem: OpenCode Cannot Start or Crashes

Error messages:
```
OpenCode failed to start
MCP server connection failed
Unknown command
Signal interrupt
```

### Causes

- MCP servers not running or misconfigured
- Missing Python dependencies
- Agent Mail server not accessible
- Serena/uvx not installed or outdated
- Invalid configuration in `opencode.json`
- Incompatible model configuration

### Solutions

#### Verify OpenCode Installation

```bash
# Check OpenCode version
opencode --version

# Should output something like: opencode version X.Y.Z

# Update OpenCode if needed
curl -fsSL https://opencode.ai/install | bash
```

#### Verify Agent Mail is Running

```bash
# Check if Agent Mail server is accessible
curl http://localhost:8765/health

# Should respond with: {"status":"ok"}

# If not running, start it (in separate terminal)
am

# Check logs
tail -f ~/.agent-mail/server.log
```

#### Verify MCP Server Configuration

```bash
# Test Agent Mail MCP server directly
python -m mcp_agent_mail.server &

# Should show: MCP server started on port 5000

# Kill with: pkill -f "mcp_agent_mail"
```

#### Install/Update Serena

```bash
# Install uv if not present
curl -LsSf https://astral.sh/uv/install.sh | sh

# Test Serena
uvx --from git+https://github.com/oraios/serena serena --help

# Should show Serena help text
```

#### Validate OpenCode Configuration

```bash
# Check opencode.json syntax
python -c "import json; json.load(open('opencode.json'))"

# Should not produce errors

# Common issues to check:
# - All JSON is valid (no trailing commas)
# - All file paths exist
# - All required fields are present
# - No circular references in instructions
```

#### Debug MCP Server Startup

```bash
# Start OpenCode with verbose logging
opencode --debug

# This will show MCP server connection attempts
# Look for error messages indicating which server failed

# Check specific MCP server command
python -m mcp_agent_mail.server

# Or Serena
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --cwd .
```

#### Fix Common Configuration Issues

**Missing `ANTHROPIC_API_KEY`:**
```bash
export ANTHROPIC_API_KEY="your-key-here"
opencode
```

**Serena timeout:**
```bash
# First run of Serena downloads dependencies - this takes time
# Run this once to cache:
uvx --from git+https://github.com/oraios/serena serena --help

# Then start OpenCode:
opencode
```

**Agent Mail connection failed:**
```bash
# Ensure Agent Mail is running
am &

# Wait a few seconds
sleep 3

# Then start OpenCode
opencode
```

### Verify All Prerequisites

```bash
# Check all required tools
echo "=== Go ==="
go version

echo "=== Python ==="
python --version

echo "=== uv ==="
uv --version

echo "=== OpenCode ==="
opencode --version

echo "=== Beads ==="
bd --version

echo "=== Docker ==="
docker --version
docker-compose --version
```

### Debugging Commands

```bash
# Get OpenCode logs (if stored)
opencode --debug 2>&1 | tee opencode-debug.log

# Check Python MCP module
python -c "import mcp_agent_mail; print(mcp_agent_mail.__version__)"

# List all MCP servers that would start
# (check these commands in opencode.json)
python -m mcp_agent_mail.server --help
uvx --from git+https://github.com/oraios/serena serena start-mcp-server --help

# Test with minimal configuration
# Create a test-opencode.json with just one MCP server
```

### Prevention

- Run `am` in a separate terminal before OpenCode
- Validate `opencode.json` syntax regularly
- Keep all tools updated
- Check `docker-compose ps` before starting OpenCode
- Review README.md and AGENTS.md for correct configuration

---

## Worktree Cleanup

### Problem: Stale Worktrees or Worktree Conflicts

Error messages:
```
Worktree path already exists
Worktree is locked
Cannot remove worktree
```

### Causes

- Session ended abnormally without cleanup
- Worktree paths left over from previous sessions
- Concurrent OpenCode sessions
- Manual file deletions without proper cleanup

### Solutions

#### List All Worktrees

```bash
# Show all worktrees
git worktree list

# Show with verbose details
git worktree list -v

# Show worktrees with their status
git worktree list --porcelain
```

#### Remove Stale Worktrees

```bash
# Remove a specific worktree
git worktree remove <path>

# Force remove if locked
git worktree remove -f <path>

# Example
git worktree remove /path/to/stale/worktree
```

#### Unlock a Locked Worktree

```bash
# If a worktree is locked, find the lock file
ls -la <worktree-path>/.git

# Remove the lock file manually if needed
rm -f <worktree-path>/.git/locked

# Or use git to unlock
git worktree lock <path> --reason "fixing lock"
git worktree unlock <path>
```

#### Clean Up Corrupted Worktrees

```bash
# If a worktree is corrupted or can't be removed normally
# First, remove from git
git worktree prune

# Then manually remove the directory
rm -rf <worktree-path>

# Verify cleanup
git worktree list
```

#### Session Cleanup Best Practices

At the end of every session, ensure cleanup:

```bash
# Run session-end command
/session-end

# Manually verify no stale worktrees
git worktree list

# Should be empty or only show main working directory
```

### Debugging Commands

```bash
# Check all worktree administrative data
cat .git/worktrees/*/locked

# Find all worktrees on disk
find . -name ".git" -type d | grep worktree

# Check worktree logs
git worktree list --verbose

# Simulate worktree prune (dry run)
git worktree prune --dry-run
```

### Prevention

- Always run `/session-end` to properly clean up
- Don't manually delete worktree directories
- Avoid concurrent OpenCode sessions in the same repository
- Regular: `git worktree prune` in your main directory

---

## Common Errors

### Error: "Agent Mail not connected"

**Symptom:** Session commands fail with connection error

**Solution:**
```bash
# 1. Check if Agent Mail is running
curl http://localhost:8765/health

# 2. If not running, start it
am

# 3. Wait for it to initialize (3-5 seconds)
sleep 5

# 4. Verify connection
curl http://localhost:8765/health

# 5. Return to OpenCode and retry
```

### Error: "OpenCode can't find MCP servers"

**Symptom:** OpenCode starts but MCP tools unavailable

**Solution:**
```bash
# 1. Verify opencode.json exists and is valid
cat opencode.json | python -m json.tool | head -30

# 2. Ensure MCP server commands work
python -m mcp_agent_mail.server &
sleep 2
kill $!

# 3. Test Serena
uvx --from git+https://github.com/oraios/serena serena --help

# 4. Restart OpenCode
```

### Error: "Beads command not found"

**Symptom:** `bd` command doesn't exist

**Solution:**
```bash
# 1. Reinstall Beads
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# 2. Verify installation
bd --version

# 3. Initialize in project (if needed)
cd /path/to/open-swarm
bd init

# 4. Check Beads is properly initialized
ls -la .beads/
```

### Error: "File reservation conflict"

**Symptom:** Cannot reserve files - another agent holds them

**Solution:**
```bash
# 1. Check who has reservation
# Use Agent Mail inbox to see which agent has the files

# 2. Wait for TTL expiration (default 1 hour)
# Reservations automatically expire

# 3. Or coordinate with the other agent
/coordinate <other-agent> "Checking status on [file-pattern]"

# 4. In urgent cases, message them directly
# They can release with: /release
```

### Error: "PostgreSQL connection timeout"

**Symptom:** Temporal won't connect to database

**Solution:**
```bash
# 1. Check PostgreSQL is running
docker-compose ps postgresql

# 2. If down, restart
docker-compose restart postgresql

# 3. Wait for health check to pass
sleep 10

# 4. Verify connection
docker-compose exec postgresql psql -U temporal -d postgres -c "SELECT 1"

# 5. Restart Temporal
docker-compose restart temporal
```

### Error: "Temporal server unhealthy"

**Symptom:** Temporal container running but cluster health check fails

**Solution:**
```bash
# 1. Check logs
docker-compose logs temporal | tail -50

# 2. Verify PostgreSQL is healthy
docker-compose exec temporal pg_isready -h postgresql -U temporal

# 3. If not ready, wait longer
sleep 30

# 4. Check Temporal health directly
docker-compose exec temporal tctl --address temporal:7233 cluster health

# 5. If still failing, rebuild
docker-compose down -v
docker-compose up -d
sleep 40
docker-compose exec temporal tctl --address temporal:7233 cluster health
```

### Error: "git worktree: path already exists"

**Symptom:** Can't create worktree because path exists

**Solution:**
```bash
# 1. List existing worktrees
git worktree list

# 2. Remove the conflicting one
git worktree remove -f <path>

# 3. Or manually delete if corrupted
rm -rf <path>

# 4. Verify removed
git worktree list

# 5. Try operation again
```

### Error: "No tasks available"

**Symptom:** `/task-ready` shows no work but you expected tasks

**Solution:**
```bash
# 1. Check all tasks
bd list

# 2. Check for blocked tasks
bd list --json | grep -i "blocked\|depends"

# 3. Unblock dependencies
# Review which tasks must complete first

# 4. Create new task if needed
bd create "Task description" -t feature

# 5. Check Beads synced with Git
bd sync
```

---

## Debugging Commands

### General Diagnostics

```bash
# Full system status
echo "=== System ===" && \
docker-compose ps && \
echo && \
echo "=== Agent Mail ===" && \
curl -s http://localhost:8765/health && \
echo && \
echo "=== Go Build ===" && \
go build -o bin/open-swarm ./cmd/open-swarm && echo "Build OK" && \
echo && \
echo "=== Tests ===" && \
go test ./... -v | head -30
```

### Docker/Services Debugging

```bash
# Full service status
docker-compose ps -a

# Service logs (last 100 lines)
docker-compose logs --tail=100

# Specific service logs with follow
docker-compose logs -f temporal

# Check network connectivity
docker-compose exec temporal ping postgresql
docker-compose exec temporal ping -c 1 8.8.8.8

# Check DNS resolution
docker-compose exec temporal nslookup postgresql
```

### Agent Mail Debugging

```bash
# Check server health
curl -v http://localhost:8765/health

# Get server info
curl http://localhost:8765/info 2>/dev/null | python -m json.tool

# Check Agent Mail database
ls -la ~/.agent-mail/

# View recent logs
tail -100 ~/.agent-mail/server.log

# Kill and restart
pkill -9 -f "agent.*mail"
sleep 2
am
```

### OpenCode Debugging

```bash
# Version and paths
opencode --version
which opencode

# Dry run with verbose output
opencode --debug 2>&1 | head -50

# Check configuration file
cat opencode.json | python -m json.tool | head -50

# Verify MCP servers can start
python -m mcp_agent_mail.server &
SERVER_PID=$!
sleep 2
kill $SERVER_PID
```

### Beads Debugging

```bash
# Check Beads version and health
bd --version
bd list

# View raw issues
cat .beads/issues.jsonl | head -5

# Check Beads database
ls -la .beads/

# Sync with validation
bd sync

# Full issue list with details
bd list --json | python -m json.tool | head -100
```

### Go/Build Debugging

```bash
# Build with verbose output
go build -v -x -o bin/open-swarm ./cmd/open-swarm

# Run tests with verbose output
go test -v ./...

# Get code coverage
go test -cover ./...

# Run a specific test
go test -v -run TestNamePattern ./...

# Check dependencies
go mod tidy
go mod verify
```

### Network Debugging

```bash
# Check all listening ports
netstat -tuln | grep LISTEN

# macOS alternative
lsof -i -P -n | grep LISTEN

# Check specific service ports
for port in 5432 7233 8233 8765; do
  echo "Port $port:"
  lsof -i :$port || echo "  (not in use)"
done

# Test connectivity
telnet localhost 7233
nc -zv localhost 8765
```

### File Reservation Debugging

```bash
# Check Agent Mail archive for reservations
ls -la ~/.agent-mail/archive/file_reservations/

# Check Beads for file data
ls -la .beads/

# Look for reservation conflicts in logs
grep -i "reservation\|conflict" ~/.agent-mail/server.log | tail -20
```

### Worktree Debugging

```bash
# Detailed worktree status
git worktree list -v --porcelain

# Check for locked worktrees
git worktree list | grep locked

# Inspect specific worktree
ls -la <worktree-path>/.git

# Check main repo worktree data
ls -la .git/worktrees/
cat .git/worktrees/*/locked 2>/dev/null

# Clean and repair
git worktree prune
git fsck --full
```

---

## Quick Reference

### Before Starting Work

```bash
# 1. Verify services
docker-compose ps
curl http://localhost:8765/health

# 2. Start Agent Mail
am &

# 3. Check ports available
lsof -i :7233 || echo "Port 7233 free"
lsof -i :8765 || echo "Port 8765 free"

# 4. Start OpenCode
opencode
```

### Before Ending Session

```bash
# 1. Release file reservations
/release

# 2. Complete any tasks
/task-complete <task-id>

# 3. Clean worktrees
git worktree list  # Should be minimal

# 4. End session properly
/session-end

# 5. Sync Beads
bd sync
```

### Emergency Recovery

```bash
# Stop everything
docker-compose down
pkill -9 -f "agent.*mail"
pkill -9 -f "opencode"

# Clean problematic files
rm -rf ~/.agent-mail/tmp/*
git worktree prune

# Start fresh
docker-compose up -d
sleep 30
am &
sleep 2
opencode
```

---

## Getting Help

If you encounter an issue not covered here:

1. **Check logs first:**
   ```bash
   docker-compose logs | tail -50
   tail -50 ~/.agent-mail/server.log
   ```

2. **Gather diagnostic info:**
   ```bash
   # Collect everything
   docker-compose ps > diagnostic.txt
   docker-compose logs >> diagnostic.txt
   bd list --json >> diagnostic.txt
   git worktree list -v >> diagnostic.txt
   ```

3. **File a Beads issue:**
   ```bash
   bd create "Description of problem with diagnostic info"
   ```

4. **Review documentation:**
   - README.md - Overview and architecture
   - QUICKSTART.md - Getting started guide
   - AGENTS.md - Agent instructions
   - This file - Troubleshooting guide

5. **Contact team:**
   - Use `/coordinate` to message agents
   - Check Agent Mail inbox for responses
