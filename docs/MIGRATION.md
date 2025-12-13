# Migration Guide: v6.0.0 to v6.1.0

**Version:** v6.1.0
**Release Date:** December 12, 2025
**Compatibility:** Backward compatible with existing SDK-based workflows

---

## Table of Contents

1. [Overview](#overview)
2. [What's New](#whats-new)
3. [Breaking Changes](#breaking-changes)
4. [Backward Compatibility](#backward-compatibility)
5. [Migration Steps](#migration-steps)
6. [Configuration Updates](#configuration-updates)
7. [New Features Guide](#new-features-guide)
8. [Troubleshooting](#troubleshooting)
9. [FAQ](#faq)

---

## Overview

Version 6.1.0 introduces **Temporal workflow orchestration** as an opt-in enhancement to the existing SDK-based Reactor. This release adds powerful distributed execution capabilities while maintaining full backward compatibility with v6.0.0 workflows.

### What Changed at a Glance

| Aspect | v6.0.0 | v6.1.0 |
|--------|--------|--------|
| **Execution Model** | Local SDK-based orchestration | SDK + optional Temporal workflows |
| **Workflow Types** | Single task execution | TCR + DAG patterns |
| **Distribution** | Single machine | Cluster-capable (optional) |
| **Breaking Changes** | N/A | None - fully backward compatible |
| **New Binaries** | `open-swarm` | + `temporal-worker`, `reactor-client` |
| **Docker Compose** | Optional | Includes Temporal cluster (optional) |

---

## What's New

### Major Features

#### 1. Temporal Workflow Orchestration

**Location:** `internal/temporal/`

Distributed workflow execution for complex task orchestration:

- **Test-Commit-Revert (TCR) Workflow** - Tight-loop development pattern with automated testing and commits
- **DAG Workflow** - Parallel task execution with dependency resolution
- Complete Temporal SDK integration (v1.38.0)
- Activity layer with serializable I/O types

#### 2. Activity Layer

Three core activities for infrastructure operations:

- **Shell Activities** (`activities_shell.go`) - Execute arbitrary shell commands via bitfield/script
- **Cell Lifecycle** (`activities_cell.go`) - Bootstrap, execute, and teardown isolated cells
- **Serializable Types** - All activities use serializable input/output for Temporal's persistence

#### 3. Temporal Cluster Infrastructure

**Location:** `docker-compose.yml`

Complete Temporal ecosystem for production deployments:

```yaml
services:
  temporal:       # Temporal Server with gRPC API
  temporal-ui:    # Web UI for workflow monitoring (localhost:8080)
  postgres:       # Persistent event store
```

#### 4. New CLI Binaries

**Temporal Worker** (`cmd/temporal-worker/main.go`)
- Registers all workflows and activities
- Processes tasks from Temporal queues
- Supports 50 concurrent activities
- Auto-heartbeating for long-running tasks

**Reactor Client** (`cmd/reactor-client/main.go`)
- CLI for workflow submission
- TCR workflow submission interface
- Workflow execution tracking

#### 5. Two Workflow Patterns

**Test-Commit-Revert (TCR)**
- Execute → Test → Commit on success or Revert on failure
- Ideal for rapid development iterations
- Saga pattern for guaranteed cleanup

**Directed Acyclic Graph (DAG)**
- Multi-task workflows with dependencies
- Parallel task execution via Temporal selectors
- Toposort-based dependency resolution
- TDD integration with per-task test cycles

#### 6. New Documentation

- `docs/DAG-WORKFLOW.md` - DAG pattern implementation guide
- `docs/TCR-WORKFLOW.md` - Test-Commit-Revert pattern guide
- `docs/DEPLOYMENT.md` - Production deployment strategies
- `docs/MONITORING.md` - Observability and metrics
- Example workflow definitions

---

## Breaking Changes

**NONE.** Version 6.1.0 is fully backward compatible with v6.0.0.

### Guaranteed Stability

All existing code continues to work unchanged:

- ✅ SDK-based orchestration unchanged
- ✅ Existing `cmd/open-swarm` CLI untouched
- ✅ All infrastructure components (ports, servers, worktrees) unchanged
- ✅ OpenCode SDK integration preserved
- ✅ Existing workflow patterns continue working

**Migration strategy:** Adopt Temporal features at your own pace. Existing v6.0.0 deployments can upgrade to v6.1.0 without any changes.

---

## Backward Compatibility

### What Still Works

#### 1. Existing Orchestration

```go
// v6.0.0 code - still works in v6.1.0
portManager := infra.NewPortManager(8000, 9000)
serverManager := infra.NewServerManager()
worktreeManager := infra.NewWorktreeManager(repo, tempDir)

// SDK-based execution unchanged
client := agent.NewOpenCodeClient(port)
result, err := client.ExecutePrompt(ctx, prompt)
```

#### 2. Infrastructure Components

All core infrastructure components remain unchanged:

| Component | Location | Status |
|-----------|----------|--------|
| Port Manager | `internal/infra/ports.go` | Unchanged |
| Server Manager | `internal/infra/server.go` | Unchanged |
| Worktree Manager | `internal/infra/worktree.go` | Unchanged |
| SDK Client Wrapper | `internal/agent/client.go` | Unchanged |

#### 3. Architectural Invariants

All 6 architectural invariants from v6.0.0 remain enforced:

1. **INV-001** - Each agent runs on unique port via Port Manager
2. **INV-002** - Server working directory set to Git Worktree
3. **INV-003** - Supervisor waits for healthcheck before SDK connection
4. **INV-004** - SDK Client configured with specific BaseURL
5. **INV-005** - Server process killed on activity completion
6. **INV-006** - Command execution uses SDK `client.Command.Execute` only

#### 4. Dependencies

Existing dependencies remain:

```go
require (
    github.com/sst/opencode-sdk-go v0.19.1    // Unchanged
    gopkg.in/yaml.v3 v3.0.1                     // Unchanged
)
```

New dependencies are **purely additive**:

```go
require (
    go.temporal.io/sdk v1.38.0                  // NEW - optional
    github.com/gammazero/toposort v0.1.1        // NEW - optional
    github.com/bitfield/script v0.24.1          // NEW - optional
)
```

---

## Migration Steps

### Step 1: Upgrade Code

```bash
cd /path/to/open-swarm

# Pull latest v6.1.0 code
git fetch origin
git checkout v6.1.0

# Update Go modules (new dependencies added, existing ones preserved)
go mod download
go mod tidy
```

### Step 2: Verify Backward Compatibility

Your existing code requires **no changes**. Verify this:

```bash
# Rebuild existing binaries - should succeed without modification
go build -o bin/open-swarm ./cmd/open-swarm

# Run existing tests
go test ./internal/infra/...        # Port, server, worktree tests
go test ./internal/agent/...        # SDK client tests
go test ./internal/workflow/...     # Existing workflow tests
```

### Step 3: Optional - Enable Temporal Features

Only if you want to use new Temporal capabilities:

#### 3a. Build New Binaries

```bash
# Build Temporal worker
go build -o bin/temporal-worker ./cmd/temporal-worker

# Build Reactor client (for CLI workflow submission)
go build -o bin/reactor-client ./cmd/reactor-client
```

#### 3b. Start Temporal Infrastructure

```bash
# Start Temporal cluster (PostgreSQL + Temporal Server + UI)
docker-compose up -d

# Verify Temporal is running
curl http://localhost:7233/api/v1/namespaces  # Temporal Server API
curl http://localhost:8080                      # Temporal UI (browser)
```

#### 3c. Start Worker

```bash
# In a new terminal, start Temporal worker
./bin/temporal-worker

# Expected output:
# 2025/12/12 XX:XX:XX started temporal worker
# Workflows registered: TCRWorkflow, DAGWorkflow
# Activities registered: 15 activities
```

#### 3d. Submit Test Workflow

```bash
# Submit a TCR workflow
./bin/reactor-client tcr \
    --task "TASK-001" \
    --prompt "Add a hello function to main.go"

# Monitor in Temporal UI: http://localhost:8080
```

### Step 4: Production Deployment

#### Option A: Keep v6.0.0 Workflow (Recommended for Stability)

No changes needed. Your existing SDK-based orchestration continues working:

```bash
# Your existing deployment scripts work unchanged
./bin/open-swarm  # Continues orchestrating as before
```

#### Option B: Gradually Adopt Temporal

Run both modes simultaneously:

```bash
# Terminal 1: Keep existing SDK-based orchestration
./bin/open-swarm

# Terminal 2: Run Temporal worker (for new distributed workflows)
./bin/temporal-worker
```

#### Option C: Full Migration to Temporal (Advanced)

Requires careful refactoring. See [Temporal-Specific Deployment](#temporal-specific-deployment) below.

---

## Configuration Updates

### No Required Configuration Changes

Your existing configuration works unchanged. Optional enhancements:

### Environment Variables (Optional)

```bash
# Temporal connection (defaults work for docker-compose)
export TEMPORAL_FRONTEND_ADDRESS=localhost:7233
export TEMPORAL_NAMESPACE=default
export TEMPORAL_TASK_QUEUE=default

# Worker tuning (optional)
export TEMPORAL_WORKER_CONCURRENCY=50           # Concurrent activities
export TEMPORAL_HEARTBEAT_INTERVAL=30s          # Activity heartbeat
```

### opencode.json (No Changes Required)

Existing configuration works as-is. The `cmd/open-swarm` CLI and MCP servers operate unchanged.

### docker-compose.yml (New Optional Infrastructure)

If adopting Temporal features, use the included compose file:

```bash
# View Temporal infrastructure
cat docker-compose.yml

# Services added:
# - temporal:        Workflow orchestration server
# - temporal-ui:     Web dashboard for monitoring
# - postgres:        Event sourcing database
```

---

## New Features Guide

### Using TCR Workflow (Test-Commit-Revert)

#### When to Use

- Rapid development cycles (seconds to minutes per iteration)
- Features with automated tests
- Tight feedback loops needed
- Safe rollback on test failure desired

#### Implementation

**Via CLI (Easiest):**

```bash
# Start worker and Temporal infrastructure first
docker-compose up -d
./bin/temporal-worker &

# Submit TCR workflow
./bin/reactor-client tcr \
    --task "FEAT-123" \
    --prompt "Implement user authentication"
```

**Via Go Code:**

```go
import "open-swarm/internal/temporal"

// Execute TCR workflow
workflowID := "tcr-auth-001"
runID, err := client.ExecuteWorkflow(
    ctx,
    options.WithWorkflowID(workflowID),
    temporal.TCRWorkflow,
    temporal.TCRWorkflowInput{
        CellID: "primary",
        TaskID: "FEAT-123",
        Prompt: "Implement user authentication",
    },
)
```

**Behavior:**

1. **Execute** - Run prompt via OpenCode SDK
2. **Test** - Execute test suite automatically
3. **Commit** - On test pass, commit changes to Git
4. **Revert** - On test fail, reset --hard to previous state
5. **Repeat** - Loop for up to N iterations

See `/home/lewis/src/open-swarm/docs/TCR-WORKFLOW.md` for complete guide.

### Using DAG Workflow (Parallel Tasks)

#### When to Use

- Multi-task workflows with dependencies
- Parallel execution desired (build + test simultaneously)
- Complex task relationships
- Sequential pipelines needed

#### Example: Build and Test Parallel

```json
{
  "tasks": [
    {
      "id": "build",
      "type": "shell",
      "command": "go build ./...",
      "dependencies": []
    },
    {
      "id": "lint",
      "type": "shell",
      "command": "golangci-lint run",
      "dependencies": []
    },
    {
      "id": "test",
      "type": "shell",
      "command": "go test ./...",
      "dependencies": ["build", "lint"]
    }
  ]
}
```

See `/home/lewis/src/open-swarm/docs/DAG-WORKFLOW.md` for complete guide and examples.

### Monitoring Temporal Workflows

#### Web Dashboard

```
http://localhost:8080
```

Features:
- View all workflows and executions
- Task success/failure rates
- Timeline visualization
- Event sourcing logs
- Activity metrics

#### Command Line

```bash
# List running workflows
temporal workflow list

# Get workflow details
temporal workflow describe --workflow-id tcr-auth-001

# View events
temporal workflow show --workflow-id tcr-auth-001
```

#### Programmatic Monitoring

```go
import "go.temporal.io/sdk/client"

// Connect to Temporal
c, _ := client.Dial(client.Options{})
defer c.Close()

// Get workflow execution
execution, _ := c.DescribeWorkflowExecution(ctx, "tcr-auth-001", "")

// Access metadata
fmt.Printf("Status: %v\n", execution.WorkflowExecutionInfo.Status)
fmt.Printf("Started: %v\n", execution.WorkflowExecutionInfo.StartTime)
```

---

## Troubleshooting

### "Command 'temporal-worker' not found"

**Problem:** New binaries not built

**Solution:**

```bash
go build -o bin/temporal-worker ./cmd/temporal-worker
go build -o bin/reactor-client ./cmd/reactor-client
```

### "Connection refused: localhost:7233"

**Problem:** Temporal Server not running

**Solution:**

```bash
# Start Temporal infrastructure
docker-compose up -d

# Verify Temporal started
docker-compose ps
# Should show: temporal, temporal-ui, postgres (all running)

# Check logs
docker-compose logs temporal
```

### "Workflow execution failed: activities not registered"

**Problem:** Worker not running or activities not registered

**Solution:**

```bash
# Ensure worker is running
./bin/temporal-worker

# Should output:
# Workflows registered: TCRWorkflow, DAGWorkflow
# Activities registered: [list of activities]

# In new terminal, submit workflow
./bin/reactor-client tcr --task TASK-001 --prompt "test"
```

### "dial: unknown network localhost:8080"

**Problem:** Trying to connect to Temporal UI before it starts

**Solution:**

```bash
# Give containers time to start
docker-compose up -d
sleep 10

# Verify services running
docker-compose ps

# Access UI
curl http://localhost:8080
```

### Existing Tests Failing After Upgrade

**Problem:** Rare if upgrading correctly, but possible if modifying core code

**Solution:**

```bash
# Run tests for unchanged components first
go test ./internal/infra/...        # Port, server, worktree
go test ./internal/agent/...        # SDK client

# These should all pass with zero changes

# If failing, verify:
# 1. Go version: go version (should be 1.25+)
# 2. Dependencies: go mod tidy && go mod download
# 3. Git state: git status (should be clean)
```

### Performance: Workflows Very Slow

**Likely Cause:** First-time infrastructure startup

**Solution:**

```bash
# Temporal has cold-start overhead (~30-60s first run)
# This is normal and expected

# Subsequent workflows execute faster as cache warms

# For development, keep worker running in background:
./bin/temporal-worker > /tmp/worker.log 2>&1 &
```

---

## FAQ

### Q1: Do I Need to Use Temporal?

**No.** Temporal is completely optional. Your v6.0.0 workflows work unchanged in v6.1.0.

**Decision tree:**

- Need local, single-machine orchestration? → Keep using v6.0.0 workflow (no change needed)
- Want distributed execution across machines? → Adopt Temporal features (v6.1.0)
- Want DAG-based multi-task workflows? → Use DAG Workflow (new in v6.1.0)
- Need rapid development cycles with auto-rollback? → Use TCR Workflow (new in v6.1.0)

### Q2: Can I Run Both SDK and Temporal Workflows?

**Yes.** Both modes work simultaneously:

```bash
# Terminal 1: v6.0.0 SDK-based orchestration (unchanged)
./bin/open-swarm

# Terminal 2: v6.1.0 Temporal worker (new)
./bin/temporal-worker
```

Each system operates independently. Choose which workflows to execute in which system.

### Q3: What Are the Resource Requirements?

**For v6.0.0 workflows (unchanged):**
- Same as before: ~2GB RAM + 1-2 CPU cores per agent

**For Temporal features (new, optional):**
- Temporal Server: ~2GB RAM, 1-2 CPU cores
- PostgreSQL: ~1GB RAM (default)
- Worker process: ~500MB RAM
- Per activity: Same as agents (~2GB RAM, 1-2 cores)

**Total for full stack:** ~5-6GB RAM, 4-8 CPU cores

### Q4: Is This a Major Version Bump?

**No.** v6.1.0 is a minor version release because:
- ✅ No breaking changes
- ✅ Fully backward compatible
- ✅ New features are opt-in
- ✅ Existing APIs unchanged

Breaking changes would trigger a v7.0.0 release.

### Q5: What About Database Migrations?

**No migrations needed.** PostgreSQL is only used by Temporal for event sourcing, not by your application logic.

If using Temporal, PostgreSQL is automatically initialized by docker-compose.

### Q6: Can I Deploy to Kubernetes?

**Yes, two options:**

1. **Keep v6.0.0 workflow** - Deploy as-is, no Kubernetes changes
2. **Add Temporal** - See `docs/DEPLOYMENT.md` for Kubernetes operator patterns

### Q7: How Do I Monitor Everything?

**Monitoring options:**

1. **v6.0.0 workflows** - Existing logging unchanged
2. **Temporal workflows** - Use Temporal UI: http://localhost:8080
3. **See `docs/MONITORING.md`** - Comprehensive observability guide

### Q8: What If I Hit a Bug in v6.1.0?

**Options:**

1. **Temporary:** Downgrade to v6.0.0 (fully supported, no breaking changes)
2. **Report:** File issue with minimal reproducible example
3. **Workaround:** Disable Temporal features, stick with v6.0.0 workflow

```bash
# Downgrade if needed
git checkout v6.0.0
go build -o bin/open-swarm ./cmd/open-swarm
./bin/open-swarm  # Works identically to before
```

### Q9: Should I Upgrade Now or Wait?

**Recommended:** Upgrade when you:

1. ✅ Need distributed workflow execution
2. ✅ Want to try new TCR/DAG patterns
3. ✅ Are comfortable with optional dependencies
4. ✅ Have time for integration testing

**Don't upgrade if:**

1. Current v6.0.0 meets all your needs
2. Your team is in production and avoiding changes
3. You prefer proven, battle-tested code

**Safe either way:** No breaking changes, downgrade anytime.

### Q10: What's the Upgrade Path v6.1.0 → v7.0.0?

**Unknown at this time.** v7.0.0 is not yet planned.

Based on CHANGELOG roadmap:
- v6.2.0: Metrics export, Web UI
- v7.0.0: Kubernetes operator, auto-scaling, multi-region

When v7.0.0 releases, migration guide will be provided (likely with breaking changes, hence major version).

---

## Appendix: Implementation Details

### Singleton Pattern for Non-Serializable State

Location: `internal/temporal/globals.go`

Temporal requires all workflow/activity inputs/outputs to be serializable (JSON). Non-serializable components (port manager, server manager, worktrees) are managed as singletons:

```go
// globals.go - managed outside Temporal
var (
    portManager     *infra.PortManager       // Local to main process
    serverManager   *infra.ServerManager     // Local to main process
    worktreeManager *infra.WorktreeManager   // Local to main process
)

// Activities use these singletons
func ExecuteShellActivity(ctx context.Context, input ShellInput) (string, error) {
    // Access non-serializable managers via globals
    port := portManager.Next()
    server := serverManager.Get(port)
    worktree := worktreeManager.Get(cell.ID)

    // Execute via SDK
    return executeViaSDK(server.Client, input.Command)
}
```

**Why this approach:** Temporal's event sourcing requires all data to be serializable. Local resource management (ports, processes) cannot be event-sourced. Solution: manage locally via singletons, pass serializable results through workflows.

### Serializable Types for Temporal Persistence

All Temporal activities use serializable input/output:

```go
// ✅ Good - serializable
type ShellInput struct {
    CellID  string `json:"cell_id"`
    Command string `json:"command"`
}

type ShellOutput struct {
    ExitCode int    `json:"exit_code"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
}

// ❌ Bad - not serializable for Temporal
type Activity struct {
    Server *OpenCodeServer  // Can't serialize
    Client *OpenCodeClient  // Can't serialize
}
```

All activities in v6.1.0 follow the serializable pattern.

### Toposort-Based DAG Resolution

Location: `internal/temporal/workflow_dag.go`

DAG workflows resolve task dependencies using topological sorting:

```go
// Input: DAG with tasks and dependencies
// Output: Execution order ensuring all dependencies execute first

// Example:
tasks := []Task{
    {ID: "build", Deps: []string{}},
    {ID: "lint", Deps: []string{}},
    {ID: "test", Deps: []string{"build", "lint"}},
}

// Toposort produces execution order:
// Phase 1 (parallel): build, lint
// Phase 2 (parallel): test (after build + lint complete)
```

This enables both parallelization and correct sequencing.

---

## Next Steps

1. **Read CHANGELOG.md** - Detailed technical changes
2. **Review docs/ARCHITECTURE.md** - How v6.1.0 integrates
3. **Try TCR Workflow** - See `docs/TCR-WORKFLOW.md`
4. **Try DAG Workflow** - See `docs/DAG-WORKFLOW.md`
5. **Production Deployment** - See `docs/DEPLOYMENT.md`
6. **Monitoring** - See `docs/MONITORING.md`

---

## Support

- **Issues:** File with minimal reproducible example
- **Questions:** Review relevant documentation file
- **Bug Reports:** Include v6.1.0, your OS, and reproduction steps
- **Downgrade:** `git checkout v6.0.0` if blocking your work
