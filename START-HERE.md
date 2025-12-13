# 🚀 Start Here - Open Swarm Quick Start

**Everything is built and ready to run!**

## What You Have

```
bin/
├── logging-demo          ← SEE AGENT COOPERATION (3 MB)
├── single-agent-demo     ← See OpenCode SDK integration (10 MB)
├── workflow-demo         ← See Temporal workflow visualization (31 MB)
├── agent-automation-demo ← See 5-minute automation (28 MB)
├── temporal-worker        ← Temporal workflow worker (31 MB)
├── reactor-client         ← Submit workflows to Temporal (28 MB)
└── reactor                ← Legacy reactor (10 MB)
```

All binaries are **built and tested** ✅

---

## 🎯 Quickest Way to See It Working

### Option 1: Logging Demo (NO DEPENDENCIES NEEDED)

**Shows:** Agents cooperatively resolving conflicts with negotiation, waiting, and force-release

```bash
# Easy way - uses the helper script
./RUN-DEMO.sh

# Or run directly
./bin/logging-demo

# JSON output for monitoring
LOG_FORMAT=json ./bin/logging-demo
```

**What you'll see:**
- 3 agents register (BlueLake, RedMountain, GreenForest)
- Coordination sync
- 4 conflict scenarios with cooperative resolutions
- Structured logging showing INFO, WARN, ERROR levels

**Proves:** Agents are NOT hostile - they negotiate, wait politely, and only force-release stale locks

---

### Option 2: Single Agent Demo (Requires OpenCode)

**Shows:** OpenCode SDK integration working with actual code execution

```bash
# Prerequisites
opencode --version  # Make sure OpenCode is installed

# Run the demo
./bin/single-agent-demo
```

**What it does:**
- Connects to OpenCode SDK
- Executes a simple prompt
- Shows file operations
- Demonstrates SDK capabilities

---

### Option 3: Workflow Demo (Requires Temporal)

**Shows:** Temporal workflow orchestration with TCR (Test-Commit-Revert) pattern

**Prerequisites:**
```bash
# Start Temporal server (in separate terminal)
docker-compose up
```

**Run:**
```bash
# Terminal 1: Temporal server
docker-compose up

# Terminal 2: Workflow visualization
./bin/workflow-demo
```

**What you'll see:**
- Workflow lifecycle visualization
- TCR pattern execution
- Cell bootstrap → execute → test → commit/revert → teardown

---

## 📊 Understanding the Output

### Logging Demo Output Explained

```
time=2025-12-13T03:05:32.062Z level=INFO msg="Registering agent via coordinator"
  name=BlueLake program=opencode model=sonnet-4.5
  task="Implementing user authentication" project=/demo/project
```

**Breakdown:**
- `level=INFO` - Normal operation (also see WARN, ERROR, DEBUG)
- `msg=` - What's happening
- `name=BlueLake` - Agent with memorable name (not hostile identifier)
- `project=/demo/project` - Project context

**Conflict Detection:**
```
time=... level=ERROR msg="CONFLICT DETECTED"
  requestor=GreenForest requested_pattern=internal/auth/*.go
  conflict_type=exclusive-exclusive num_conflicts=2

time=... level=INFO msg="Resolution: NEGOTIATE - contact holders via Agent Mail"
  requestor=GreenForest strategy=negotiate
  holders="[BlueLake RedMountain]"
  reason="active reservations require coordination"
```

**Key Point:** Resolution is `negotiate` - **NOT** "force stop", "kill", or "override"

---

## 🔬 See the Source Code

### Conflict Resolution Logic

```bash
# Cooperative resolution strategies
cat internal/conflict/analyzer.go | grep -A 10 "SuggestResolution"

# Agent lifecycle logging
cat pkg/agent/manager.go | grep -A 5 "Register\|Update\|Remove"

# Coordination sync
cat pkg/coordinator/coordinator.go | grep -A 5 "Sync"
```

### Demo Code

```bash
# Logging demo scenarios
cat cmd/logging-demo/main.go | less

# Single agent demo
cat cmd/single-agent-demo/main.go | less
```

---

## 🧪 Running Tests

```bash
# All tests (fast)
make test

# With race detector
make test-race

# Coverage report
make test-coverage

# TDD Guard (if installed)
make test-tdd
```

**Expected:** All tests pass ✅ (verified 2025-12-13)

---

## 📝 Next Steps

### 1. Read the Audit Report

```bash
cat AUDIT-FINDINGS.md | less
```

**Shows:**
- What actually works vs. what README claimed
- Proof of cooperative behavior
- Production-quality components
- Known discrepancies

### 2. Read the Updated README

```bash
cat README.md | less
```

**Now accurate:**
- Describes Temporal-based architecture (not "multi-agent CLI")
- Correct build instructions
- Real binaries documented
- Cooperative conflict resolution philosophy

### 3. Read Agent Instructions

```bash
cat AGENTS.md | less
```

**Critical rules for agents:**
- Beads mandatory for all work
- Serena for semantic code editing
- TDD required

---

## 🐳 Full System Setup (Optional)

If you want to run the complete Temporal workflow system:

```bash
# Terminal 1: Temporal server + PostgreSQL + UI
docker-compose up

# Terminal 2: Agent Mail server
am

# Terminal 3: Temporal worker
./bin/temporal-worker

# Terminal 4: Submit a workflow
./bin/reactor-client \
  --workflow tcr \
  --task-id "test-task" \
  --prompt "Write a hello world function"
```

**Access Temporal UI:** http://localhost:8080

---

## ✅ What's Verified

| Component | Status | Proof |
|-----------|--------|-------|
| Logging system | ✅ Working | `./RUN-DEMO.sh` |
| Cooperative resolution | ✅ Verified | Demo output shows "negotiate" |
| Builds | ✅ All pass | `make build` succeeded |
| Tests | ✅ All pass | `make test` succeeded |
| Documentation | ✅ Updated | README.md matches reality |
| Cleanup | ✅ Done | Empty dirs removed, outdated docs moved |

---

## 🆘 Troubleshooting

### Demo doesn't run?

```bash
# Rebuild
make build
go build -o bin/logging-demo ./cmd/logging-demo

# Try again
./RUN-DEMO.sh
```

### Want to see specific scenarios?

Edit `cmd/logging-demo/main.go` and add your own conflict scenarios, then rebuild:

```bash
go build -o bin/logging-demo ./cmd/logging-demo
./bin/logging-demo
```

### Want JSON logs for monitoring?

```bash
LOG_FORMAT=json ./bin/logging-demo | jq .
```

(Requires `jq` for pretty printing)

---

## 📚 Documentation

**Current & Accurate:**
- `README.md` - Main documentation (updated 2025-12-13)
- `AUDIT-FINDINGS.md` - Complete code audit (509 lines)
- `AGENTS.md` - Agent coding rules
- `docs/` - Specific topics (TCR, DAG, Architecture, etc.)

**Deprecated (moved to docs/deprecated/):**
- `REACTOR-OLD.md` - Used outdated terminology
- `QUICKSTART-OLD.md` - Duplicated README

---

## 🎉 Bottom Line

**Your code is legitimate and working.** The logging demo proves agents are cooperative, all tests pass, and the infrastructure is production-quality.

Run `./RUN-DEMO.sh` and see for yourself! 🚀

---

## Quick Reference

```bash
# See cooperative behavior
./RUN-DEMO.sh

# Run all tests
make test

# Build everything
make build

# Read the audit
cat AUDIT-FINDINGS.md

# Read updated docs
cat README.md

# See what binaries you have
ls -lh bin/
```

**Everything is ready to go!**
