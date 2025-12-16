# TCR Benchmark - Quick Start Guide

## Current Status ✅

**Infrastructure is working!** 

- ✅ Automated startup/shutdown scripts created
- ✅ Temporal + PostgreSQL running
- ✅ OpenCode server running on port 3000
- ✅ Temporal worker running
- ✅ Simplified workflows (no worktrees) implemented
- ✅ Basic TCR workflow executing successfully
- ⚠️  OpenCode not writing files to disk (needs configuration fix)

## What Works Right Now

The entire infrastructure starts cleanly and workflows execute. The OpenCode LLM is being called successfully and responding, but files aren't being written to disk yet.

## Quick Start

### 1. Start Everything (Fully Automated)

```bash
./scripts/start-benchmark-infra.sh
```

This will:
- Clean up any zombie processes
- Start Temporal + PostgreSQL via Docker
- Start OpenCode server on port 3000
- Build all binaries
- Start Temporal worker
- Save PIDs for clean shutdown

### 2. Run a Benchmark

```bash
# Basic TCR (single-shot execution)
./bin/simple-benchmark \
  -strategy basic \
  -runs 3 \
  -prompt "Implement FizzBuzz"

# Enhanced TCR (6-gate process with LLM validation)
./bin/simple-benchmark \
  -strategy enhanced \
  -runs 3 \
  -prompt "Implement FizzBuzz"
```

### 3. Monitor Progress

- **Temporal UI**: http://localhost:8081
- **Worker logs**: `tail -f worker.log`
- **OpenCode logs**: `tail -f opencode.log`

### 4. Stop Everything (Fully Automated)

```bash
./scripts/stop-benchmark-infra.sh
```

This will:
- Stop Temporal worker
- Stop OpenCode server
- Stop Docker services
- Clean up zombie processes
- Remove temporary git branches
- Clean up PID files

## Architecture (Simplified)

### No More Worktrees! 🎉

The complex worktree/cell isolation has been removed. Now we use:

- **Git branches** for isolation (not worktrees)
- **Single OpenCode server** (not multiple spawned servers)
- **Direct repository access** (no file copying)
- **Standard git operations** (simpler, faster)

### Workflow Flow

**Basic TCR:**
```
Create Branch → Execute LLM Prompt → Run Tests → Commit/Revert → Cleanup
```

**Enhanced TCR:**
```
Create Branch
  → Gate 1: Generate Tests (LLM)
  → Gate 2: Lint Tests
  → Gate 3: Verify RED (tests fail)
  → Gate 4: Generate Implementation (LLM)
  → Gate 5: Verify GREEN (tests pass)
  → Gate 6: Multi-Review (LLM validation)
  → Commit/Revert
  → Cleanup
```

## Current Issue & Next Steps

### Issue
OpenCode is responding to prompts but not writing files to disk. The workflows complete but there's nothing to commit.

### Likely Causes
1. OpenCode session needs explicit file write configuration
2. Working directory may not be set correctly
3. May need to use specific OpenCode tools/commands

### Fix Options

**Option A: Configure OpenCode SDK to write files**
- Pass working directory to session
- Enable file write tools explicitly
- Use specific agents that write files

**Option B: Parse OpenCode response and write files ourselves**
- Extract code from LLM response
- Parse file paths from response
- Write files manually in the activity

**Option C: Use OpenCode commands instead of prompts**
- Use `/write` or `/edit` commands
- More explicit about file operations
- Better control over file system changes

## Files Created

### Infrastructure Scripts
- `scripts/start-benchmark-infra.sh` - Automated startup
- `scripts/stop-benchmark-infra.sh` - Automated shutdown

### Simplified Implementation
- `internal/temporal/activities_simple.go` - Activities without worktrees
- `internal/temporal/workflow_simple_basic.go` - Basic TCR workflow
- `internal/temporal/workflow_simple_enhanced.go` - Enhanced TCR workflow
- `cmd/simple-benchmark/main.go` - CLI benchmark runner

### Documentation
- `SIMPLE_BENCHMARK.md` - Detailed setup guide
- `QUICKSTART.md` - This file

## Command Reference

### Start/Stop Infrastructure
```bash
./scripts/start-benchmark-infra.sh    # Start everything
./scripts/stop-benchmark-infra.sh     # Stop everything
```

### Run Benchmarks
```bash
# Basic strategy
./bin/simple-benchmark -strategy basic -runs 3 -prompt "Your task"

# Enhanced strategy
./bin/simple-benchmark -strategy enhanced -runs 3 -prompt "Your task"

# With custom OpenCode URL
./bin/simple-benchmark -strategy basic -runs 1 -prompt "Task" -opencode-url http://localhost:8080
```

### Manual Control
```bash
# Build
make build

# Start worker manually
./bin/temporal-worker

# Start OpenCode manually
opencode serve --port 3000

# Start/stop Docker
make docker-up
make docker-down
```

### Cleanup
```bash
# Remove temporary branches
git branch -D tcr-basic-*
git branch -D tcr-enhanced-*

# View logs
tail -f worker.log
tail -f opencode.log
```

## Theory Being Tested

**Hypothesis**: Enhanced TCR (with LLM-as-Judge gates) produces more reliable code than Basic TCR

**Expected Results:**
- **Basic TCR**: ~40-60% success rate, faster execution (~1-2 min/run)
- **Enhanced TCR**: ~80-100% success rate, slower execution (~3-5 min/run)

**Trade-off**: Speed vs Reliability

## Monitoring & Debugging

### Check Service Status
```bash
# Temporal
docker ps | grep temporal

# Worker
pgrep -f temporal-worker

# OpenCode
curl http://localhost:3000/health
```

### View Workflow Details
1. Open http://localhost:8081
2. Find workflow by ID: `simple-bench-{strategy}-{timestamp}-{run}`
3. View execution history
4. Check activity inputs/outputs
5. See retry attempts and errors

### Common Issues

**OpenCode connection refused**
```bash
# Check if running
pgrep -f opencode

# Restart
pkill -9 opencode
opencode serve --port 3000
```

**Worker not processing**
```bash
# Check logs
tail -f worker.log

# Restart
pkill -9 temporal-worker
./bin/temporal-worker > worker.log 2>&1 &
```

**Heartbeat timeout**
- LLM calls can take 30-120 seconds
- Timeouts are set to 2 minutes
- If timing out, increase in workflow files

## Success! 🎉

The infrastructure is fully automated and working. The remaining task is to configure OpenCode to actually write files to disk, which is a configuration/API usage issue rather than an architectural problem.

All the complex worktree/cell/server management has been removed, making the system much simpler to debug and maintain.