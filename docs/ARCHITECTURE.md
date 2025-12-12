# Open Swarm Architecture

**Version:** 6.0.0
**Scale:** ENTERPRISE
**Model:** SDK-Driven Reactor with Bare-Metal Isolation

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Core Components](#core-components)
3. [Workflow Execution Flow](#workflow-execution-flow)
4. [Activity Lifecycle](#activity-lifecycle)
5. [Cell Bootstrap Sequence](#cell-bootstrap-sequence)
6. [DAG Resolution & Execution](#dag-resolution--execution)
7. [Architectural Invariants](#architectural-invariants)
8. [Directory Structure](#directory-structure)
9. [Data Flow Diagrams](#data-flow-diagrams)
10. [Deployment Patterns](#deployment-patterns)

---

## System Overview

Open Swarm is an enterprise-grade multi-agent orchestration system that enables multiple isolated AI agents to execute tasks in parallel without conflicts.

### The Architecture Trinity

```
┌─────────────────────────────────────────────────────────┐
│              REACTOR SUPERVISOR                          │
│         (Go Application / Temporal Client)               │
└──────────────┬──────────────────────────────────────────┘
               │
        ┌──────┴───────┬──────────────┬──────────────┐
        ▼              ▼              ▼              ▼
    ┌──────┐       ┌──────┐       ┌──────┐       ┌──────┐
    │ CELL │       │ CELL │       │ CELL │  ...  │ CELL │
    │  #1  │       │  #2  │       │  #3  │       │  #N  │
    └──────┘       └──────┘       └──────┘       └──────┘
     Port 8000      Port 8001      Port 8002      Port 800X

Each Cell Contains:
  ├─ Git Worktree (Isolated filesystem)
  ├─ OpenCode Server (localhost:PORT)
  └─ SDK Client (HTTP/REST connection)
```

### Key Design Principles

1. **Isolation First** - Each agent operates in its own Git worktree and OpenCode server
2. **Process Independence** - No shared process state across cells
3. **Port Multiplexing** - Unique ports (8000-9000) enable parallel server instances
4. **Health-Aware Bootstrap** - Servers must pass healthchecks before SDK connection
5. **Saga Pattern** - Guaranteed cleanup via deferred teardown activities
6. **Type Safety** - Workflow activities use serializable input/output types only

---

## Core Components

### 1. The Supervisor: Temporal Client

**Location:** `cmd/reactor/main.go`, `cmd/temporal-worker/main.go`

The Supervisor coordinates agent execution through Temporal workflows:

```go
// Bootstrap phase
portManager := infra.NewPortManager(8000, 9000)      // INV-001
serverManager := infra.NewServerManager()            // INV-002, INV-003
worktreeManager := infra.NewWorktreeManager(...)     // Isolation layer

// Execute workflow
client.ExecuteWorkflow(ctx, TCRWorkflowInput{
    CellID:      "primary",
    TaskID:      "TASK-001",
    Prompt:      "Implement feature X",
})
```

**Responsibilities:**
- Spawn Temporal workers (`cmd/temporal-worker`)
- Manage port allocation (1000 available: 8000-9000)
- Control Git worktree lifecycle
- Monitor cell health and timeouts
- Execute recovery strategies (R3, RB, IG)

**Max Capacity:** 50 concurrent agents per machine
- Limited by PortManager: 1000 ports ÷ 50 agents = 20 ports/agent buffer
- Limited by system resources (CPU, memory, file descriptors)

---

### 2. The Brain: OpenCode Server

**Location:** Each cell runs: `opencode serve --port X --dir ./worktrees/Y`

Each agent gets its own isolated OpenCode instance:

```bash
# Cell-1
opencode serve --port 8000 --hostname localhost --dir ./worktrees/cell-primary-1733925600

# Cell-2
opencode serve --port 8001 --hostname localhost --dir ./worktrees/cell-secondary-1733925601

# Cell-N
opencode serve --port 800X --hostname localhost --dir ./worktrees/cell-N-173392560Y
```

**Invariants Enforced:**
- INV-002: Working directory is set to Git worktree (not shared repo)
- INV-003: Healthcheck (200 OK on /health) before SDK connection

**Lifecycle:**
```
Boot → Healthcheck Loop → Ready → SDK Operations → Shutdown
(cold start: 1-2s)  (200ms intervals, 10s timeout)
```

---

### 3. The Nerve: OpenCode Go SDK

**Location:** `internal/agent/client.go`

Type-safe SDK wrapper with reactor-specific integration:

```go
// Create SDK client
client := opencode.NewClient(
    option.WithBaseURL("http://localhost:8000"),  // INV-004
)

// Execute operations
result, err := client.Session.Prompt(ctx, sessionID, params)
```

**Enforces:**
- INV-004: SDK configured with specific BaseURL (localhost:PORT)
- INV-006: Command execution via SDK only (`client.Command.Execute`)

**Supported Operations:**
- `Session.Prompt()` - Send AI prompts
- `Command.Execute()` - Run shell commands
- `File.Status()` - Get modified files
- `File.Read()` / `File.Write()` - File operations

---

### 4. The Hand: Git Worktrees

**Location:** `internal/infra/worktree.go`

Independent Git worktrees provide true filesystem isolation:

```
Repository (main)
├── main branch
├── worktrees/
│   ├── cell-primary-1733925600/     ← Cell-1
│   │   ├── .git (shared, worktree-specific refs)
│   │   ├── main.go
│   │   └── [modified by Cell-1]
│   ├── cell-secondary-1733925601/   ← Cell-2
│   │   ├── .git (shared, worktree-specific refs)
│   │   ├── main.go
│   │   └── [modified by Cell-2]
│   └── cell-N-173392560Y/           ← Cell-N
│       ├── .git (shared, worktree-specific refs)
│       ├── main.go
│       └── [modified by Cell-N]
```

**Benefits:**
- Each worktree is on `main` but has independent state
- Changes in Cell-1 don't affect Cell-2
- Can test in parallel without conflicts
- Clean test-commit-revert semantics

**Cleanup:**
```bash
git worktree prune        # Remove stale entries
rm -rf ./worktrees/*      # Delete worktree directories
```

---

## Workflow Execution Flow

### High-Level Workflow Pipeline

```
User/CLI Input
    ↓
[Reactor Supervisor]
    ├─ Parse task (ID, description, prompt)
    ├─ Execute Temporal Workflow
    │  (TCRWorkflow or TddDagWorkflow)
    └─ Monitor completion
         ↓
    [Temporal Worker Pool]
         ├─ Route to BootstrapCell activity
         ├─ Route to ExecuteTask activity
         ├─ Route to RunTests activity
         ├─ Route to Commit/Revert activity
         └─ Route to TeardownCell activity
         ↓
    Result (Success/Failure)
```

### Two Primary Workflows

#### 1. TCR Workflow (Test-Commit-Revert)

**File:** `internal/temporal/workflows_tcr.go`

**Purpose:** Single-task execution with test validation

```
┌────────────────────────────────────────────────────┐
│           TCRWorkflow(TCRWorkflowInput)             │
└────────────────────────────────────────────────────┘
  │
  ├─► [BOOTSTRAP PHASE]
  │     └─ BootstrapCell(cellID, branch)
  │        ├─ Allocate port 8000-9000
  │        ├─ Create Git worktree
  │        ├─ Start opencode serve
  │        ├─ Healthcheck loop (200ms, 10s timeout)
  │        └─ Initialize SDK client
  │
  ├─► [EXECUTE PHASE]
  │     └─ ExecuteTask(bootstrap, taskInput)
  │        ├─ Create OpenCode session
  │        ├─ Send prompt via SDK
  │        ├─ Agent modifies files
  │        └─ Retrieve modified file list
  │
  ├─► [TEST PHASE]
  │     └─ RunTests(bootstrap)
  │        ├─ SDK: client.Command.Execute("go", "test", "./...")
  │        ├─ Parse test results
  │        └─ Return testsPassed boolean
  │
  ├─► [DECISION POINT]
  │     └─ IF testsPassed THEN
  │            ├─ CommitChanges(bootstrap, message)
  │            │  └─ git commit -m "Task TASK-001: ..."
  │            └─ RETURN success
  │        ELSE
  │            ├─ RevertChanges(bootstrap)
  │            │  └─ git reset --hard HEAD
  │            └─ RETURN failure
  │
  └─► [TEARDOWN PHASE] (deferred, always runs)
       └─ TeardownCell(bootstrap)
          ├─ Kill opencode serve (process group)
          ├─ Remove Git worktree
          └─ Release port

        SAGA PATTERN: Cleanup guaranteed even on failure
```

**Timeouts & Retries:**

| Component | Timeout | Retry | Backoff |
|-----------|---------|-------|---------|
| Healthcheck | 10s | 1 | 200ms interval |
| Activity | 10min | 1 (non-idempotent) | None |
| Teardown | 2min | 3 | Exponential |

---

#### 2. TDD DAG Workflow (Directed Acyclic Graph)

**File:** `internal/temporal/workflows_dag.go`

**Purpose:** Multi-task orchestration with dependency resolution

```
┌────────────────────────────────────────────────────┐
│        TddDagWorkflow(DAGWorkflowInput)             │
│  Retries entire DAG until success or manual abort  │
└────────────────────────────────────────────────────┘
  │
  ├─► [ATTEMPT LOOP] ◄────────────────┐
  │    (1, 2, 3, ...)                  │
  │     │                               │
  │     ├─► [DAG PHASE]                │
  │     │    ├─ Build dependency graph │
  │     │    │  (from Task.Deps)       │
  │     │    │                         │
  │     │    ├─ Topological Sort       │
  │     │    │  Input: Task edges      │
  │     │    │  Output: flatOrder[]    │
  │     │    │                         │
  │     │    ├─ Parallel Execution     │
  │     │    │  ├─ For each task:      │
  │     │    │  │  IF all deps done:   │
  │     │    │  │   └─ Launch activity │
  │     │    │  │      (RunScript)     │
  │     │    │  └─ Use selector to     │
  │     │    │     wait for completion │
  │     │    │                         │
  │     │    └─ Handle failures:       │
  │     │       └─ Abort DAG, raise    │
  │     │          error              │
  │     │                              │
  │     ├─► IF all tasks passed        │
  │     │    └─ RETURN success         │
  │     │       (break retry loop)     │
  │     │                              │
  │     └─► IF tasks failed            │
  │          ├─ Log failure/attempt    │
  │          ├─ Wait for signal:       │
  │          │  "FixApplied"          │
  │          │  (human intervention)   │
  │          └─ Loop back to ATTEMPT ──┘
  │             (increment attempt)
```

**DAG Example:**

```go
tasks := []Task{
    {Name: "build", Command: "go build ./...", Deps: []string{}},
    {Name: "lint", Command: "golangci-lint run", Deps: []string{"build"}},
    {Name: "test", Command: "go test ./...", Deps: []string{"build"}},
    {Name: "coverage", Command: "go test -cover ./...", Deps: []string{"test"}},
}

// Computed order: [build, lint, test, coverage]
// Lint and test run in parallel after build
// Coverage waits for test
```

**Dependency Graph Visualization:**

```
       build
       /    \
    lint    test
            /
       coverage
    (no sync points, max parallelism)
```

---

## Activity Lifecycle

### Activity Phases & State Transitions

```
┌──────────────────────────────────────────────────────────────┐
│                      ACTIVITY LIFECYCLE                      │
└──────────────────────────────────────────────────────────────┘

START ACTIVITY
    │
    ├─► [INIT PHASE]
    │    ├─ Deserialize Input (JSON)
    │    ├─ Validate parameters
    │    └─ Initialize state
    │
    ├─► [EXECUTION PHASE]
    │    ├─ RecordHeartbeat(ctx, "status")
    │    │  (every 30s for >30s activities)
    │    ├─ Perform work
    │    ├─ RecordHeartbeat(ctx, "progress")
    │    └─ Handle errors (non-recoverable abort)
    │
    ├─► [RESULT PHASE]
    │    ├─ Serialize Output (JSON)
    │    ├─ Return (output, error)
    │    └─ Temporal records result
    │
    └─► COMPLETE
```

### Activity Options

```go
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Minute,    // Max execution time
    HeartbeatTimeout:    30 * time.Second,    // Max time without heartbeat
    ScheduleToCloseTimeout: 15 * time.Minute, // Max total time
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    1 * time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    30 * time.Second,
        MaximumAttempts:    3,
    },
}
ctx = workflow.WithActivityOptions(ctx, ao)
```

---

## Cell Bootstrap Sequence

### Sequential Startup Flow

```
START: BootstrapCell(cellID="primary", branch="main")
│
├─────────────────────────────────────────────────────────┐
│ PHASE 1: PORT ALLOCATION (INV-001)                      │
├─────────────────────────────────────────────────────────┤
│
│  portManager.Allocate()
│    └─ Lock global port set
│    └─ Find unused port in [8000, 9000]
│    └─ Mark port as allocated
│    └─ Return port (e.g., 8000)
│    └─ Defer: Release port on error
│
├─────────────────────────────────────────────────────────┐
│ PHASE 2: WORKTREE CREATION                              │
├─────────────────────────────────────────────────────────┤
│
│  worktreeManager.CreateWorktree(worktreeID, branch)
│    ├─ worktreeID = "cell-primary-1733925600"
│    ├─ Compute path: ./worktrees/{worktreeID}
│    ├─ Execute: git worktree add {path} {branch}
│    │  (clones shared .git, checks out worktree-specific refs)
│    ├─ Verify path exists
│    ├─ Return Worktree{ID, Path}
│    └─ Defer: Remove worktree on error
│
├─────────────────────────────────────────────────────────┐
│ PHASE 3: SERVER BOOT (INV-002, INV-003)                │
├─────────────────────────────────────────────────────────┤
│
│  serverManager.BootServer(worktreePath, port)
│    │
│    ├─► Construct command:
│    │    cmd = exec.CommandContext(ctx,
│    │        "opencode", "serve",
│    │        "--port", "8000",
│    │        "--hostname", "localhost"
│    │    )
│    │
│    ├─► Set working directory:
│    │    cmd.Dir = "./worktrees/cell-primary-1733925600"  [INV-002]
│    │
│    ├─► Configure process group:
│    │    cmd.SysProcAttr.Setpgid = true
│    │    (enables group kill for cleanup)
│    │
│    ├─► Start server:
│    │    cmd.Start()
│    │    pid = cmd.Process.Pid
│    │
│    ├─► HEALTHCHECK LOOP [INV-003]:
│    │    │
│    │    ├─ healthCtx, cancel := context.WithTimeout(ctx, 10s)
│    │    ├─ ticker := time.NewTicker(200ms)
│    │    │
│    │    └─ for {
│    │        select {
│    │        case <-healthCtx.Done():
│    │          // Timeout: kill server, return error
│    │          sm.killProcess(cmd)
│    │          return nil, "failed to become ready"
│    │
│    │        case <-ticker.C:
│    │          // Poll /health endpoint
│    │          resp, _ := client.Get("http://localhost:8000/health")
│    │          if resp.StatusCode == 200:
│    │            ready = true
│    │            return ServerHandle{...}, nil
│    │        }
│    │      }
│    │
│    └─ Defer: Kill server on error
│
├─────────────────────────────────────────────────────────┐
│ PHASE 4: SDK CLIENT INIT (INV-004)                      │
├─────────────────────────────────────────────────────────┤
│
│  client := agent.NewClient(baseURL, port)
│    └─ baseURL = "http://localhost:8000"  [INV-004]
│    └─ client.sdk = opencode.NewClient(option.WithBaseURL(...))
│    └─ No API key needed for local connections
│
├─────────────────────────────────────────────────────────┐
│ PHASE 5: RETURN SERIALIZED BOOTSTRAP                    │
├─────────────────────────────────────────────────────────┤
│
│  return &BootstrapOutput{
│    CellID:       "primary",
│    Port:         8000,
│    WorktreeID:   "cell-primary-1733925600",
│    WorktreePath: "./worktrees/cell-primary-1733925600",
│    BaseURL:      "http://localhost:8000",
│    ServerPID:    12345,
│  }
│
└─────────────────────────────────────────────────────────┘

SUCCESS: Cell fully bootstrapped and ready for ExecuteTask
```

### Timing Characteristics

| Phase | Duration | Notes |
|-------|----------|-------|
| Port allocation | <1ms | RwMutex lock |
| Worktree creation | 50-200ms | `git worktree add` |
| Server boot | 1-2s | `opencode serve` startup |
| Healthcheck | 200-2000ms | 10s timeout, 200ms polling |
| SDK init | <1ms | Just object creation |
| **Total** | **~2-4s** | Per cell bootstrap |

### Failure Modes & Recovery

| Failure | Detection | Recovery | Code |
|---------|-----------|----------|------|
| Port exhausted | `Allocate()` returns error | Fail activity (R3 retry) | `BOOT_RETRY` |
| Worktree exists | `git worktree add` fails | Prune stale, retry (R3) | `R3` |
| Server won't start | Exec fails | Fail activity | `BOOT_RETRY` |
| Healthcheck timeout | 10s elapsed | Kill process, fail activity | `BOOT_RETRY` |
| Server crash before health | Exec.Wait() error | Detected at health check | `BOOT_RETRY` |

---

## DAG Resolution & Execution

### Graph Construction Algorithm

```
Input: []Task{
    {Name: "A", Command: "...", Deps: []},
    {Name: "B", Command: "...", Deps: []},
    {Name: "C", Command: "...", Deps: [B]},
    {Name: "D", Command: "...", Deps: [A, C]},
}

STEP 1: BUILD TASK MAP
────────────────────
taskMap := {
    "A": Task{...},
    "B": Task{...},
    "C": Task{...},
    "D": Task{...},
}

STEP 2: BUILD EDGES
──────────────────
edges := []
For each task:
    For each dep:
        edges += Edge{dep, task}

Result:
edges := [
    {from: B, to: C},    ← C depends on B
    {from: A, to: D},    ← D depends on A
    {from: C, to: D},    ← D depends on C
]

STEP 3: TOPOLOGICAL SORT
────────────────────────
input:  edges = [B→C, A→D, C→D]
algo:   Kahn's algorithm (in-degree based)
output: flatOrder = [A, B, C, D]

Verification:
  ✓ A has no deps     → can start first
  ✓ B has no deps     → can start first (parallel with A)
  ✓ C depends on B    → must wait for B
  ✓ D depends on A,C  → must wait for both

STEP 4: CYCLE DETECTION
──────────────────────
If edges contain cycle (e.g., A→B→C→A):
    return error "cycle detected in DAG"
    (prevents infinite loops)
```

### Parallel Execution Strategy

```
TIME ──────────────────────────────────────► (horizontal axis)

Attempt 1:
│
├─ A ─────────┐
│             │
├─ B ─────────┤
│             ├─ D ──────────┐
│             │              │
└─ C ────────────────────────┘
│                  │
└──────────────────┘
 (selector waits for any completion)

Task States:
{
    "A": {status: running, future: f1},
    "B": {status: running, future: f2},
    "C": {status: pending},
    "D": {status: pending},
}

When A completes:
{
    "A": {status: completed, output: "..."},
    "B": {status: running, future: f2},
    "C": {status: runnable},      ← C now has all deps
    "D": {status: pending},
}
 → Launch C immediately

When B completes:
{
    "A": {status: completed},
    "B": {status: completed},
    "C": {status: running, future: f3},
    "D": {status: pending},
}

When C completes:
{
    "A": {status: completed},
    "B": {status: completed},
    "C": {status: completed},
    "D": {status: runnable},      ← All deps done
}
 → Launch D

When D completes:
 → All tasks done, DAG succeeds
```

### Selector-Based Waiting Pattern

```go
// Efficient async waiting in Temporal workflows
selector := workflow.NewSelector(ctx)

for taskName, taskFuture := range pendingFutures {
    selector.AddFuture(taskFuture, func(f workflow.Future) {
        var output string
        err := f.Get(ctx, &output)

        if err != nil {
            failedTasks = append(failedTasks, taskName)
        } else {
            completed[taskName] = true
        }

        delete(pendingFutures, taskName)
    })
}

// Blocks until ANY future completes
selector.Select(ctx)
// Wakes up, processes completion, loops back
```

**Why not goroutines?**
- Temporal workflows can't use OS goroutines
- Selector is workflow-native: deterministic, replay-safe
- Single-threaded event loop in Temporal SDK

---

## Architectural Invariants

Six immutable laws enforced by the architecture:

| ID | Invariant | Layer | Enforcement | Failure Mode |
|----|-----------|----|------------|--------------|
| **INV-001** | Each Agent runs `opencode serve` on unique port | PortManager | `Allocate()` locks, reserves 8000-9000 | Port exhaustion → fail activity, retry R3 |
| **INV-002** | Server working directory = Git Worktree path | ServerManager | `cmd.Dir = worktreePath` in BootServer | Files edited in wrong place → test failure |
| **INV-003** | Supervisor waits for Server Healthcheck (200 OK) | ServerManager | Healthcheck loop: 200ms polling, 10s timeout | SDK connects before ready → connection refused |
| **INV-004** | SDK Client configured with BaseURL (localhost:PORT) | agent.Client | `option.WithBaseURL()` in NewClient | SDK connects to wrong server or shared repo |
| **INV-005** | Server Process killed when Activity completes | ServerManager | Process group termination: `syscall.Kill(-pgid, SIGTERM)` | Zombie processes → port leaks, resource exhaustion |
| **INV-006** | Command execution uses SDK `client.Command.Execute` | agent.Client | Workflow activities call SDK only | Direct shell execution → untracked changes |

### Enforcement Mechanisms

**INV-001: Port Uniqueness**
```go
type PortManager struct {
    mu        sync.RWMutex
    allocated map[int]bool
    min, max  int
}

func (pm *PortManager) Allocate() (int, error) {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    for p := pm.min; p <= pm.max; p++ {
        if !pm.allocated[p] {
            pm.allocated[p] = true
            return p, nil
        }
    }
    return 0, errors.New("no ports available")
}
```

**INV-002: Worktree Isolation**
```go
cmd := exec.CommandContext(ctx, "opencode", "serve",
    "--port", fmt.Sprintf("%d", port),
    "--hostname", "localhost",
)
cmd.Dir = worktreePath  // ← INV-002: Set to worktree, not repo root
```

**INV-003: Healthcheck Before SDK**
```go
healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
for {
    select {
    case <-healthCtx.Done():
        sm.killProcess(cmd)
        return nil, "healthcheck timeout"
    case <-ticker.C:
        resp, _ := client.Get(baseURL + "/health")
        if resp.StatusCode == 200 {
            return ServerHandle{...}, nil  // Only return when healthy
        }
    }
}
```

**INV-004: SDK BaseURL Configuration**
```go
// ✓ Correct
sdk := opencode.NewClient(option.WithBaseURL("http://localhost:8000"))

// ✗ Wrong (would connect to wrong server)
sdk := opencode.NewClient(option.WithBaseURL("http://repo-host:5000"))
```

**INV-005: Process Group Termination**
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // Create new process group
}

// Kill entire group (server + children)
syscall.Kill(-pgid, syscall.SIGTERM)
```

**INV-006: SDK-Only Command Execution**
```go
// ✓ Correct: Use SDK
result, _ := client.GetSDK().Command.Execute(ctx, opencode.CommandExecuteParams{
    Command: opencode.F("go test ./..."),
})

// ✗ Wrong: Direct shell access
cmd := exec.CommandContext(ctx, "go", "test", "./...")
cmd.Run()
```

---

## Directory Structure

```
open-swarm/
├── cmd/
│   ├── reactor/                  # Main orchestrator entry point
│   │   └── main.go              # Supervisor (Temporal client driver)
│   ├── temporal-worker/          # Temporal worker process
│   │   └── main.go              # Worker registration, queue listening
│   ├── reactor-client/           # Future: client library
│   │   └── main.go              # (placeholder)
│   └── open-swarm/              # Legacy CLI tool
│       └── main.go              # (multi-agent coordination)
│
├── internal/
│   ├── infra/                    # Infrastructure layer
│   │   ├── ports.go             # Port allocation (INV-001)
│   │   ├── ports_test.go
│   │   ├── server.go            # Server lifecycle (INV-002, 003, 005)
│   │   └── worktree.go          # Git worktree management
│   │
│   ├── agent/                    # Agent layer
│   │   ├── client.go            # OpenCode SDK wrapper (INV-004, 006)
│   │   ├── types.go             # TaskContext, PromptOptions, etc.
│   │   └── manager.go           # Agent identity management
│   │
│   ├── workflow/                 # Workflow activities
│   │   ├── activities.go        # Bootstrap, Execute, Test, Commit/Revert
│   │   └── types.go             # CellBootstrap, TaskInput/Output
│   │
│   ├── temporal/                 # Temporal workflows
│   │   ├── workflows_tcr.go     # Test-Commit-Revert workflow
│   │   ├── workflows_dag.go     # TDD DAG workflow
│   │   ├── activities_cell.go   # Cell lifecycle activities
│   │   ├── activities_shell.go  # Shell command activities
│   │   ├── globals.go           # Singleton managers
│   │   └── types.go             # Serializable types
│   │
│   └── config/                   # Configuration
│       └── config.go            # Settings, validation
│
├── pkg/                          # Public API packages
│   ├── coordinator/             # Multi-agent coordination
│   │   └── coordinator.go
│   ├── agent/                   # Agent management
│   │   └── manager.go
│   └── tasks/                   # Task management
│       └── tasks.go
│
├── internal/                     # (duplicate, see above)
│
├── tests/                        # Integration tests
│   └── (test files)
│
├── docs/
│   ├── ARCHITECTURE.md          # This file
│   └── API.md                   # Future API documentation
│
├── .beads/                      # Beads issue tracking
│   └── issues.jsonl            # Git-committed issues
│
├── .opencode/                   # OpenCode configuration
│   ├── tool/                   # Custom MCP tools
│   ├── command/                # Slash commands
│   ├── agent/                  # Custom agent definitions
│   └── plugin/                 # Plugins
│
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── Makefile                     # Build & test targets
├── docker-compose.yml           # Local dev environment
├── opencode.json               # OpenCode config
├── AGENTS.md                   # Agent instructions
├── REACTOR.md                  # Reactor documentation
├── README.md                   # Project overview
└── QUICKSTART.md              # Quick start guide
```

### Key Interfaces

**`infra.PortManager`**
- `Allocate() (int, error)` - Get unused port
- `Release(port int) error` - Return port
- `IsAvailable(port int) bool` - Check availability

**`infra.ServerManager`**
- `BootServer(ctx, path, id, port) (*ServerHandle, error)` - Start server
- `Shutdown(handle *ServerHandle) error` - Stop server
- `killProcess(cmd *exec.Cmd) error` - Kill process group

**`infra.WorktreeManager`**
- `CreateWorktree(id, branch) (*Worktree, error)` - Create isolated checkout
- `RemoveWorktree(id) error` - Clean up worktree
- `CleanupAll() error` - Remove all stale worktrees

**`agent.Client`**
- `ExecutePrompt(ctx, prompt, opts) (*PromptResult, error)` - Send prompt
- `ExecuteCommand(ctx, cmd) (*CommandResult, error)` - Run command
- `GetFileStatus(ctx) (*FileStatus, error)` - Get modified files

**`workflow.Activities`**
- `BootstrapCell(ctx, id, branch) (*CellBootstrap, error)`
- `ExecuteTask(ctx, cell, task) (*TaskOutput, error)`
- `RunTests(ctx, cell) (bool, error)`
- `CommitChanges(ctx, cell, msg) error`
- `RevertChanges(ctx, cell) error`
- `TeardownCell(ctx, cell) error`

---

## Data Flow Diagrams

### Request Flow: Reactor → Supervisor → Worker

```
┌──────────────────┐
│  User/CLI Tool   │
│                  │
│ Calls reactor    │
│ --task TASK-001  │
│ --prompt "..."   │
└────────┬─────────┘
         │
         ▼
┌──────────────────────────────────┐
│   Reactor (Supervisor)           │
│   (cmd/reactor/main.go)          │
│                                  │
│ 1. Parse CLI args                │
│ 2. Create Temporal client        │
│ 3. ExecuteWorkflow(TCRWorkflow)  │
└────────┬─────────────────────────┘
         │
         │ Temporal Protocol
         │ (gRPC to localhost:7233)
         ▼
┌──────────────────────────────────┐
│   Temporal Server                │
│   (localhost:7233)               │
│                                  │
│ - Receives WorkflowExecutionStart│
│ - Queues activities on           │
│   "reactor-task-queue"           │
└────────┬─────────────────────────┘
         │
         │ Task Queue
         ▼
┌──────────────────────────────────┐
│   Temporal Worker Pool           │
│   (cmd/temporal-worker/main.go)  │
│                                  │
│ - Polls: GetActivity()           │
│ - Dequeues: BootstrapCell        │
│ - Executes: Activity             │
│ - Polls: GetActivity()           │
│ - Dequeues: ExecuteTask          │
│ - Executes: Activity             │
│ ... (continue for all activities)│
│ - Polls: GetActivity()           │
│ - Dequeues: TeardownCell         │
│ - Executes: Activity             │
│ - Completes: WorkflowCompletion  │
└────────┬─────────────────────────┘
         │
         │ Temporal Protocol
         │ (workflow result)
         ▼
┌──────────────────────────────────┐
│   Reactor (Supervisor)           │
│                                  │
│ - Receives: TCRWorkflowResult    │
│ - Prints: Success/Failure        │
│ - Exits with code 0 or 1         │
└──────────────────────────────────┘
```

### File Modification Flow

```
User Intent
    │
    ▼
┌─────────────────────────────────────┐
│  ExecuteTask Activity               │
│  (internal/temporal/activities_cell)│
└──────────┬──────────────────────────┘
           │
           ├─► Reconstruct Cell
           │   from BootstrapOutput
           │
           ├─► Extract SDK Client
           │   baseURL, port
           │
           ├─► Build Prompt
           │   with task context
           │
           ├─► HTTP POST to OpenCode
           │   /session/prompt
           │   (inside worktree)
           │
           ▼
      ┌─────────────────────────────┐
      │  OpenCode Server            │
      │  (opencode serve --port X)  │
      │  CWD: worktrees/cell-N      │
      └──────────┬──────────────────┘
                 │
                 ├─► LLM API Call
                 │   (Claude, etc.)
                 │
                 ├─► LLM suggests edits
                 │
                 ├─► Apply edits to
                 │   files on disk
                 │   (in worktree only)
                 │
                 └─► Return file list
                     to executor
           │
           │
           ▼
┌─────────────────────────────────────┐
│  Back in ExecuteTask Activity       │
│                                     │
│ SDK returns: PromptResult           │
│ {                                   │
│   FilesModified: [                 │
│     "main.go",                      │
│     "main_test.go",                │
│   ],                                │
│   Output: "Added feature X"         │
│ }                                   │
│                                     │
│ Return TaskOutput                   │
└──────────┬──────────────────────────┘
           │
           │ Serialized via Temporal
           │
           ▼
     Workflow receives TaskOutput
```

### Worktree Isolation: Parallel Cell Modifications

```
Time →
                                 Reactor Supervisor
                            /        |        \
                           /         |         \
                    Cell-1         Cell-2     Cell-3
                    Port 8000      Port 8001  Port 8002
                   (Agent A)      (Agent B)  (Agent C)
                          │            │          │
                  ┌───────┴───────┬────┴────┬────┴──────┐
                  ▼               ▼         ▼            ▼
Repository       Repository      Repository Repository   Repository
(main)           (shared .git)    (shared .git) (shared.git) (shared .git)
│                │                │            │           │
├─ main          ├─ ./worktrees/cell-1        ├─ main-Agent-A...
│   ├─ api.go    │   ├─ api.go    ← edited    │   └─ HEAD → cell-1
│   ├─ main.go   │   ├─ main.go   (different) │
│   └─ test.go   │   └─ test.go               │
│                │                │            │           │
│                ├─ ./worktrees/cell-2        ├─ main-Agent-B...
│                │   ├─ api.go                │   └─ HEAD → cell-2
│                │   ├─ main.go   ← edited    │
│                │   └─ test.go   (different) │
│                │                │            │           │
│                └─ ./worktrees/cell-3        └─ main-Agent-C...
│                    ├─ api.go                    └─ HEAD → cell-3
│                    ├─ main.go   ← edited
│                    └─ test.go   (different)

Invariant: Each worktree has independent .git/HEAD
           pointing to cell-specific reflog, so changes
           don't interfere
```

---

## Deployment Patterns

### Single Machine (Vertical Scaling)

```
Physical Machine
├─ CPU: 4-8 cores
├─ RAM: 8-32 GB
└─ Disk: 50+ GB

Resources per Agent Cell:
├─ Process: ~1 opencode server instance (~200MB RAM)
├─ Worktree: ~1 isolated Git checkout (~100MB-1GB code)
├─ Port: 1 unique port from 8000-9000
└─ Execution: ~30s-5min per task

Max Concurrent Agents: 50 (configurable)
├─ Port limit: 1000 available ÷ 50 = 20 buffer per agent
├─ Memory estimate: 50 × 250MB = 12.5GB
├─ CPU utilization: LLM time-bound, not CPU-bound
└─ File descriptors: 50 agents × 10 fds ≈ 500 (plenty)

Recommendation:
├─ Start with MaxAgents = 20 for safety
├─ Monitor memory & port usage
└─ Scale to 50 only after tuning
```

### Multi-Machine (Horizontal Scaling)

```
┌──────────────────────────────────────────┐
│         Load Balancer / Queue             │
│    (Redis, SQS, RabbitMQ, Kafka)         │
└──────────┬─────────────────────┬─────────┘
           │                     │
    ┌──────▼────────┐    ┌──────▼────────┐
    │   Reactor-1   │    │   Reactor-2   │
    │  (Host A)     │    │  (Host B)     │
    │               │    │               │
    │ 50 cells max  │    │ 50 cells max  │
    │ Port 8000-    │    │ Port 8000-    │
    │ 9000          │    │ 9000          │
    └───────┬───────┘    └───────┬───────┘
            │                    │
            └────────┬───────────┘
                     │
         ┌───────────▼──────────────┐
         │   Shared Git Repository  │
         │  (GitHub, GitLab, etc.)  │
         │   NFS/EFS mount          │
         └──────────────────────────┘

Coordination:
├─ Load balancer assigns tasks to least-loaded reactor
├─ Each reactor runs independent worker pool
├─ Shared Git repo allows cross-machine worktrees
├─ No direct communication between reactors
└─ Queue backend tracks distributed state

Scaling Benefits:
├─ Linear: N reactors = N × 50 = max agents
├─ Fault isolation: Reactor-1 failure ≠ Reactor-2 failure
├─ Geographic distribution: Reactors near compute resources
└─ Cost efficiency: Pay for used capacity only
```

### Docker/Kubernetes Deployment

```yaml
# docker-compose.yml (single host)
version: '3.8'
services:
  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgres
    depends_on:
      - postgres

  postgres:
    image: postgres:14
    environment:
      POSTGRES_PASSWORD: temporal
      POSTGRES_USER: temporal
      POSTGRES_DB: temporal
    volumes:
      - postgres_data:/var/lib/postgresql/data

  reactor-worker:
    build: .
    command: /app/cmd/temporal-worker/main
    environment:
      - TEMPORAL_HOST_PORT=temporal:7233
      - REPO_DIR=/repo
      - WORKTREE_BASE=/worktrees
    volumes:
      - /path/to/git/repo:/repo
      - /tmp/worktrees:/worktrees
    depends_on:
      - temporal
    deploy:
      replicas: 3  # 3 worker instances × 50 agents = 150 max
      resources:
        limits:
          cpus: '2.0'
          memory: 4G

  reactor-supervisor:
    build: .
    command: /app/cmd/reactor/main --max-agents 50
    environment:
      - TEMPORAL_HOST_PORT=temporal:7233
      - REPO_DIR=/repo
      - WORKTREE_BASE=/worktrees
    volumes:
      - /path/to/git/repo:/repo
      - /tmp/worktrees:/worktrees
    depends_on:
      - temporal
    deploy:
      resources:
        limits:
          cpus: '4.0'
          memory: 8G

volumes:
  postgres_data:
```

---

## Silent Killers & Mitigations

### 1. Server Cold Start

**Problem:** `opencode serve` takes 1-2s to boot; SDK might connect before ready.

**Mitigation:** INV-003 healthcheck loop
```go
healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
for {
    select {
    case <-healthCtx.Done():
        return nil, "timeout"
    case <-ticker.C:
        resp, _ := client.Get(baseURL + "/health")
        if resp.StatusCode == 200 {
            return handle, nil  // Only proceed when ready
        }
    }
}
```

### 2. Token/Cost Visibility

**Problem:** SDK abstracts LLM calls; hidden token usage can cause surprise bills.

**Solution:** Time-boxing as cost control
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)  // Hard limit
defer cancel()
```

**Future:** Parse SDK response headers for token usage
```go
resp, _ := client.Session.Prompt(ctx, ...)
tokenCount := resp.Usage.InputTokens + resp.Usage.OutputTokens
costUSD := float64(tokenCount) * (0.003 / 1_000_000)
```

### 3. Zombie Processes

**Problem:** If Reactor crashes, `opencode` processes remain, consuming ports.

**Mitigation:** INV-005 process group termination
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // Create new process group
}

// Kill entire group
syscall.Kill(-pgid, syscall.SIGTERM)
```

**Additional safeguard:** Periodic cleanup script
```bash
# Kill all stale opencode processes
pkill -f "opencode serve"

# Prune stale worktrees
git worktree prune

# Remove orphaned worktree directories
find ./worktrees -type d -mtime +1 -exec rm -rf {} \;
```

### 4. Port Leaks

**Problem:** Ports allocated but not released due to panic/crash.

**Mitigation:** Deferred cleanup in activities
```go
port, _ := pm.Allocate()
defer pm.Release(port)  // Always releases, even on panic
```

**Detection:** Monitor port availability
```go
available, _ := pm.CountAvailable()
if available < 10 {
    log.Warn("Low port availability", "remaining", available)
}
```

### 5. DAG Deadlock

**Problem:** Circular dependencies or missing tasks cause DAG to stall forever.

**Mitigation:** Cycle detection + stall detection
```go
// Cycle detection (topological sort)
_, err := toposort.Toposort(edges)
if err != nil {
    return fmt.Errorf("cycle detected: %w", err)
}

// Stall detection
if len(pendingFutures) == 0 && len(completed) < len(tasks) {
    return fmt.Errorf("DAG stalled - no tasks runnable")
}
```

---

## Performance Characteristics

### Latency (per task)

| Phase | Duration | Notes |
|-------|----------|-------|
| Bootstrap | 2-4s | Port alloc + worktree + server + healthcheck |
| Execute | 30s-5min | Depends on prompt complexity |
| Test | 5-30s | `go test ./...` duration |
| Commit/Revert | 1-2s | Git operation |
| Teardown | 2-5s | Kill server + remove worktree |
| **End-to-end** | **~1-10 min** | Typical task |

### Throughput (parallel agents)

| Config | Agents | Tasks/Hour | Notes |
|--------|--------|-----------|-------|
| 1 cell | 1 | ~6-12 | Sequential |
| 10 cells | 10 | ~60-120 | Parallel |
| 50 cells | 50 | ~300-600 | Max single machine |
| N reactors (horizontal) | N×50 | N×300-600 | Cluster |

### Resource Utilization

| Resource | Per Cell | 50 Cells | Limit |
|----------|----------|----------|-------|
| Memory | 200-300MB | 10-15GB | System RAM |
| Disk (worktree) | 100MB-1GB | 5-50GB | SSD space |
| Ports | 1 | 50 | 1000 available |
| File descriptors | ~10 | ~500 | OS limit (usually 1024+) |
| CPU | Idle (LLM I/O bound) | 1-4 cores active | Depends on LLM |

---

## Testing Strategy

### Unit Tests

**File:** `internal/infra/ports_test.go`

```go
func TestPortManager_Allocate(t *testing.T) {
    pm := NewPortManager(8000, 8005)

    p1, _ := pm.Allocate()  // 8000
    p2, _ := pm.Allocate()  // 8001
    p3, _ := pm.Allocate()  // 8002

    pm.Release(p2)
    p4, _ := pm.Allocate()  // 8001 (reused)

    if p4 != p2 {
        t.Fatalf("expected reuse of released port")
    }
}
```

### Integration Tests

**File:** `internal/temporal/workflows_test.go`

```bash
go test ./internal/temporal/... -v
# Tests TCR workflow end-to-end with real Temporal server
```

### Load Tests

```bash
# Spawn N simultaneous tasks
for i in {1..50}; do
    ./bin/reactor --task "TASK-$i" --prompt "..." &
done
wait

# Monitor: port usage, memory, CPU
```

---

## Monitoring & Observability

### Key Metrics

```go
type Metrics struct {
    CellBootstrapTime    time.Duration  // Per cell
    TaskExecutionTime    time.Duration
    TestPassRate         float64        // 0-100%
    PortUtilization      int            // 0-1000
    ActiveCells          int
    ZombieProcesses      int
    FailedActivities     int
    WorkflowCompletions  int
}
```

### Logging

**Structured logs from Reactor:**
```
🚀 Reactor-SDK v6.0.0 - Enterprise Agent Orchestrator
📊 Configuration:
   Repository: /home/lewis/src/open-swarm
   Worktree Base: ./worktrees
   Branch: main
   Max Agents: 50
   Port Range: 8000-9000

🔧 Initializing infrastructure...
📦 Bootstrapping agent cell...
✅ Cell bootstrapped on port 8000
📁 Worktree: ./worktrees/cell-primary-1733925600
⚙️  Executing task...
✅ Task completed successfully
🧪 Running tests...
✅ Tests passed
💾 Committing changes...
✅ Changes committed
🧹 Tearing down cell...
✅ Reactor execution complete
```

**Temporal Server Logs:**
```
WorkflowID: reactor-task-001
WorkflowType: temporal.TCRWorkflow
State: COMPLETED
Activities:
  - BootstrapCell: 3.2s
  - ExecuteTask: 2m15s
  - RunTests: 18s
  - CommitChanges: 1.5s
  - TeardownCell: 3.8s
Total: 2m42s
```

---

## Future Enhancements

### v6.1.0 (Q1 2024)

- [ ] Full `go-workflows` integration for complex DAGs
- [ ] `--parallel` flag implementation for multi-task execution
- [ ] Prometheus metrics export

### v6.2.0 (Q2 2024)

- [ ] Distributed mode with message queue (Redis/SQS)
- [ ] Web UI for monitoring cells
- [ ] Cost tracking per task (token accounting)

### v7.0.0 (H2 2024)

- [ ] Kubernetes operator for cluster deployment
- [ ] Auto-scaling based on queue depth
- [ ] Multi-region support
- [ ] Caching layer for LLM responses

---

## References

- [OpenCode Documentation](https://opencode.ai/docs/)
- [OpenCode Go SDK](https://github.com/sst/opencode-sdk-go)
- [Temporal Go SDK](https://github.com/temporalio/sdk-go)
- [Git Worktrees](https://git-scm.com/docs/git-worktree)
- [Tessl Planning Architect](https://tessl.io)

---

**Document Version:** 1.0
**Last Updated:** December 2024
**Status:** PRODUCTION
