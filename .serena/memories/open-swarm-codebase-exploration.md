# Open Swarm Codebase Exploration - Complete Analysis

## Executive Summary

**Open Swarm** is a multi-agent coordination framework for Go that enables:
- **Temporal Workflows**: DAG-based and TCR (Test-Commit-Revert) workflow patterns
- **Agent/Reactor Pattern**: Isolated "cells" (worktrees) with OpenCode servers for agent execution
- **Infrastructure Management**: Port allocation, server lifecycle, Git worktree management
- **Multi-Agent Coordination**: Agent Mail (messaging) + Beads (task tracking) integration

---

## 1. TEMPORAL WORKFLOWS

### 1.1 Workflow Types

#### **DAG Workflow** (`internal/temporal/workflows_dag.go`)
- **Purpose**: Execute tasks in dependency order with parallelism
- **Pattern**: Test-Driven Development loop with retry on failure
- **Key Features**:
  - Topological sort of task dependencies
  - Parallel task execution when dependencies allow
  - Human-in-the-loop: waits for "FixApplied" signal on failure
  - Automatic retry loop until success

**Input Structure**:
```go
type DAGWorkflowInput struct {
    WorkflowID string  // Unique workflow identifier
    Branch     string  // Git branch to use
    Tasks      []Task  // List of tasks with dependencies
}

type Task struct {
    Name    string   // Task name
    Command string   // Shell command to execute
    Deps    []string // Task dependencies
}
```

**Execution Flow**:
1. Build dependency graph from tasks
2. Topological sort to determine execution order
3. Launch tasks in parallel when dependencies met
4. Wait for task completion using Temporal selectors
5. On failure: wait for "FixApplied" signal, then retry entire DAG
6. On success: return

#### **TCR Workflow** (`internal/temporal/workflows_tcr.go`)
- **Purpose**: Single-task execution with test-driven validation
- **Pattern**: Bootstrap → Execute → Test → (Commit|Revert) → Teardown
- **Key Features**:
  - Isolated cell bootstrap (port, worktree, server)
  - Task execution via OpenCode
  - Automatic test running
  - Conditional commit/revert based on test results
  - Guaranteed cleanup via saga pattern

**Input Structure**:
```go
type TCRWorkflowInput struct {
    CellID      string // Agent cell identifier
    Branch      string // Git branch
    TaskID      string // Task identifier
    Description string // Task description
    Prompt      string // OpenCode prompt
}

type TCRWorkflowResult struct {
    Success      bool     // Overall success
    TestsPassed  bool     // Test result
    FilesChanged []string // Modified files
    Error        string   // Error message if failed
}
```

**Execution Flow**:
1. **Bootstrap**: Allocate port, create worktree, start OpenCode server
2. **Execute**: Run prompt in OpenCode cell
3. **Test**: Run test suite
4. **Commit/Revert**: Based on test results
5. **Teardown**: Clean up resources (guaranteed via defer)

### 1.2 Activity Definitions

#### **CellActivities** (`internal/temporal/activities_cell.go`)
Wraps workflow activities for Temporal serialization:

- `BootstrapCell(BootstrapInput) → BootstrapOutput`
  - Allocates port, creates worktree, starts server
  - Returns serializable output (no process pointers)

- `ExecuteTask(BootstrapOutput, TaskInput) → TaskOutput`
  - Runs OpenCode prompt in cell
  - Returns success/failure with file modifications

- `RunTests(BootstrapOutput) → bool`
  - Executes test suite in cell
  - Returns pass/fail

- `CommitChanges(BootstrapOutput, string) → error`
  - Commits changes with message

- `RevertChanges(BootstrapOutput) → error`
  - Reverts to pre-execution state

- `TeardownCell(BootstrapOutput) → error`
  - Cleans up resources

#### **ShellActivities** (`internal/temporal/activities_shell.go`)
Simple shell execution:

- `RunScript(command string) → string`
  - Executes shell command, returns output

- `RunScriptInDir(dir, command string) → string`
  - Executes in specific directory

### 1.3 Global Manager Initialization

**File**: `internal/temporal/globals.go`

```go
func InitializeGlobals(portMin, portMax int, repoDir, worktreeBase string) error
func GetManagers() (PortManagerInterface, ServerManagerInterface, WorktreeManagerInterface)
```

- Singleton pattern with sync.Once
- Initializes port, server, and worktree managers
- Used by CellActivities to access infrastructure

---

## 2. AGENT/REACTOR PATTERN

### 2.1 Infrastructure Layer (`internal/infra/`)

#### **PortManager** (`ports.go`)
Manages port allocation for OpenCode servers:

```go
type PortManager struct {
    mu        sync.Mutex
    minPort   int
    maxPort   int
    allocated map[int]bool
    nextPort  int
}

// Methods:
Allocate() (int, error)           // Get next available port
Release(port int) error           // Free a port
AllocatedCount() int              // Count allocated ports
IsAllocated(port int) bool        // Check if port in use
AvailableCount() int              // Count available ports
```

#### **ServerManager** (`server.go`)
Manages OpenCode server lifecycle:

```go
type ServerHandle struct {
    Port       int
    WorktreeID string
    WorkDir    string
    Cmd        *exec.Cmd
    BaseURL    string
    PID        int
}

type ServerManager struct {
    opencodeCommand string
    healthTimeout   time.Duration
    healthInterval  time.Duration
}

// Methods:
BootServer(ctx, worktreePath, worktreeID, port) → *ServerHandle
Shutdown(handle) → error
IsHealthy(handle) → bool
```

#### **WorktreeManager** (`worktree.go`)
Manages Git worktrees for agent isolation:

```go
type WorktreeManager struct {
    baseDir string  // Base directory for worktrees
    repoDir string  // Repository directory
}

type WorktreeInfo struct {
    ID   string
    Path string
}

// Methods:
CreateWorktree(id, branch) → *WorktreeInfo
RemoveWorktree(id) → error
ListWorktrees() → []*WorktreeInfo
PruneWorktrees() → error
CleanupAll() → error
```

### 2.2 Workflow Activities Layer (`internal/workflow/`)

#### **Activities** (`activities.go`)
High-level activity orchestration:

```go
type Activities struct {
    portManager     PortManagerInterface
    serverManager   ServerManagerInterface
    worktreeManager WorktreeManagerInterface
}

type CellBootstrap struct {
    CellID       string
    Port         int
    WorktreeID   string
    WorktreePath string
    ServerHandle *ServerHandle
    Client       *http.Client
}

// Methods:
BootstrapCell(ctx, cellID, branch) → *CellBootstrap
TeardownCell(ctx, cell) → error
ExecuteTask(ctx, cell, task) → *TaskOutput
RunTests(ctx, cell) → bool
CommitChanges(ctx, cell, message) → error
RevertChanges(ctx, cell) → error
```

### 2.3 Agent Management (`pkg/agent/`)

#### **Manager** (`manager.go`)
Tracks active agents:

```go
type Agent struct {
    Name            string
    Program         string
    Model           string
    TaskDescription string
    LastActive      time.Time
    ProjectKey      string
}

type Manager struct {
    projectKey string
    agents     map[string]*Agent
    mu         sync.RWMutex
}

// Methods:
Register(agent *Agent) error
Get(name string) → *Agent
List() → []*Agent
CountActive() → int
Remove(name string) → error
Update(agent *Agent) → error
```

---

## 3. CONFIGURATION

### 3.1 Config Structure (`internal/config/config.go`)

```go
type Config struct {
    Project      ProjectConfig
    Model        ModelConfig
    MCPServers   MCPServersConfig
    Behavior     BehaviorConfig
    Coordination CoordinationConfig
    Build        BuildConfig
}

type ProjectConfig struct {
    Name             string
    Description      string
    WorkingDirectory string
}

type ModelConfig struct {
    Default string
    Agents  map[string]string  // Agent name → model
}

type BehaviorConfig struct {
    AutoCoordinate   bool
    CheckReservations bool
    AutoRegister     bool
    PreserveThreads  bool
    UseTodos         bool
}

type CoordinationConfig struct {
    Agent        AgentConfig
    Messages     MessagesConfig
    Reservations ReservationsConfig
    Threads      ThreadsConfig
}

type BuildConfig struct {
    Commands BuildCommands
    Slots    BuildSlots
}
```

### 3.2 OpenCode Integration (`opencode.json`)

**MCP Servers Configured**:
1. **agent-mail** (remote): http://localhost:8765/mcp
   - Messaging, file reservations, agent coordination
2. **serena** (local): LSP-based semantic code navigation
3. **beads** (local): Git-backed issue tracking
4. **sequential-thinking**: Reasoning tool
5. **chrome-devtools**: Browser automation
6. **opencode-agent**: OpenCode CLI integration

**Sub-agents Defined**:
- `validator`: Deep code verification (Opus 4.5)
- `coordinator`: Multi-agent coordination (Sonnet 4.5)
- `reviewer`: Code review specialist (Sonnet 4.5)
- `tester`: Test writing specialist (Sonnet 4.5)
- `explorer`: Codebase exploration (Haiku 4.5)
- `documenter`: Documentation writer (Sonnet 4.5)

**Commands Configured**:
- `/sync`: Register with Agent Mail
- `/reserve`: Reserve files for editing
- `/release`: Release file reservations
- `/task-ready`: Check Beads for ready work
- `/task-start`: Start a Beads task
- `/task-complete`: Complete a Beads task
- `/review`: Code review
- `/explore`: Codebase exploration
- `/test`: Write tests
- `/doc`: Generate documentation
- `/build`: Build project
- `/lint`: Run linters

---

## 4. ENTRY POINTS

### 4.1 Reactor Service (`cmd/reactor/main.go`)

**Purpose**: Main orchestrator for agent execution

**Configuration**:
```go
const (
    MaxConcurrentAgents = 50
    PortRangeMin        = 8000
    PortRangeMax        = 9000
)

type Config struct {
    RepoDir      string
    WorktreeBase string
    Branch       string
    MaxAgents    int
}
```

**Command-line Flags**:
- `--repo`: Git repository directory (default: ".")
- `--worktrees`: Base directory for worktrees (default: "./worktrees")
- `--branch`: Git branch to use (default: "main")
- `--max-agents`: Maximum concurrent agents (default: 50)
- `--task`: Task ID (required)
- `--desc`: Task description
- `--prompt`: Task prompt (required)
- `--parallel`: Run tasks in parallel (not yet implemented)

**Execution Flow**:
1. Parse flags and validate inputs
2. Initialize infrastructure (port, server, worktree managers)
3. Clean up existing worktrees
4. Bootstrap cell (allocate port, create worktree, start server)
5. Execute task via OpenCode
6. Run tests
7. Commit or revert based on test results
8. Teardown cell

### 4.2 Temporal Worker (`cmd/temporal-worker/main.go`)

**Purpose**: Registers workflows and activities with Temporal server

**Responsibilities**:
- Register DAG and TCR workflows
- Register cell and shell activities
- Connect to Temporal server
- Run worker loop

### 4.3 Reactor Client (`cmd/reactor-client/main.go`)

**Purpose**: Client for submitting tasks to reactor

**Responsibilities**:
- Submit task requests
- Monitor execution status
- Retrieve results

### 4.4 Demo Binaries

- `cmd/quality-monitor/`: Quality monitoring
- `cmd/workflow-demo/`: Workflow demonstration
- `cmd/agent-automation-demo/`: Agent automation example

---

## 5. EXISTING TESTS

### 5.1 Temporal Workflow Tests (`internal/temporal/workflows_test.go`)

```go
TestDAGToposort()                      // Verify topological sort
TestDAGDependencyValidation()          // Validate dependency handling
TestTCRWorkflowInputSerialization()    // JSON serialization
TestTCRWorkflowResultSerialization()   // JSON serialization
TestDAGWorkflowInputSerialization()    // JSON serialization
TestTaskStructure()                    // Task struct validation
```

### 5.2 Infrastructure Tests

- `internal/infra/ports_test.go`: Port manager tests
- `internal/workflow/activities_test.go`: Activity tests
- `internal/temporal/activities_test.go`: Cell activity tests
- `internal/temporal/workflows_dag_test.go`: DAG workflow tests
- `internal/temporal/workflows_tcr_test.go`: TCR workflow tests

### 5.3 Configuration Tests (`internal/config/config_test.go`)

- Config loading and validation

### 5.4 Agent Manager Tests (`pkg/agent/manager_test.go`)

- Agent registration, retrieval, updates

### 5.5 Coordinator Tests (`pkg/coordinator/coordinator_test.go`)

- Multi-agent coordination

### 5.6 Integration Tests (`test/e2e_test.go`)

- End-to-end workflow execution

---

## 6. ARCHITECTURE PATTERNS

### 6.1 Dependency Injection

All managers use constructor injection:
```go
func NewActivities(pm PortManagerInterface, sm ServerManagerInterface, wm WorktreeManagerInterface) *Activities
```

### 6.2 Interface-Driven Design

All infrastructure components implement interfaces:
- `PortManagerInterface`
- `ServerManagerInterface`
- `WorktreeManagerInterface`

### 6.3 Saga Pattern

TCR workflow uses saga pattern for guaranteed cleanup:
```go
defer func() {
    // Cleanup always runs, even on error
    workflow.ExecuteActivity(disconnCtx, cellActivities.TeardownCell, bootstrap)
}()
```

### 6.4 Temporal Patterns

- **Activity Options**: Timeouts, retry policies
- **Workflow Selectors**: Wait for multiple futures
- **Signals**: Human-in-the-loop via "FixApplied" signal
- **Disconnected Context**: Cleanup independent of main workflow

---

## 7. WHAT NEEDS TO BE BUILT

### 7.1 OpenCode Integration

**Current State**: Reactor can bootstrap OpenCode servers and execute tasks

**Needed**:
1. **OpenCode Client Library**: Wrapper for HTTP API
   - Send prompts to running OpenCode server
   - Parse responses
   - Handle streaming output
   - Error handling

2. **Task Execution Engine**: 
   - Map Beads tasks to OpenCode prompts
   - Handle Agent Mail coordination
   - Track file modifications
   - Capture test results

3. **Result Aggregation**:
   - Collect output from multiple agents
   - Merge results
   - Update Beads with completion status

### 7.2 Agent Coordination

**Current State**: Agent manager tracks agents, basic coordination

**Needed**:
1. **Agent Mail Integration**:
   - Send/receive messages between agents
   - File reservation management
   - Thread-based conversation tracking

2. **Beads Integration**:
   - Create tasks from requirements
   - Update task status
   - Track dependencies
   - Link messages to tasks

3. **Conflict Resolution**:
   - Detect file conflicts
   - Coordinate access
   - Automatic retry logic

### 7.3 Workflow Enhancements

**Current State**: DAG and TCR workflows exist

**Needed**:
1. **Parallel DAG Execution**: Currently sequential
2. **Workflow Composition**: Combine multiple workflows
3. **Error Recovery**: Better error handling and recovery
4. **Monitoring**: Metrics and observability

### 7.4 Testing Infrastructure

**Current State**: Basic unit tests exist

**Needed**:
1. **Mock OpenCode Server**: For testing without real server
2. **Integration Test Fixtures**: Sample workflows and tasks
3. **Performance Tests**: Benchmark agent execution
4. **Chaos Testing**: Failure scenarios

---

## 8. WHAT NEEDS TO BE EXTENDED

### 8.1 Port Manager

**Current**: Simple sequential allocation

**Extend**:
- Port pool management
- Graceful release on server crash
- Port reuse after timeout

### 8.2 Server Manager

**Current**: Basic process management

**Extend**:
- Health check improvements
- Automatic restart on failure
- Resource limits (CPU, memory)
- Logging aggregation

### 8.3 Worktree Manager

**Current**: Basic Git worktree operations

**Extend**:
- Worktree pooling for faster startup
- Automatic cleanup of stale worktrees
- Worktree snapshots for rollback
- Concurrent worktree operations

### 8.4 Activities

**Current**: Basic cell lifecycle

**Extend**:
- Streaming output from OpenCode
- Progress tracking
- Cancellation support
- Resource monitoring

### 8.5 Configuration

**Current**: Static YAML/JSON

**Extend**:
- Environment variable substitution
- Configuration validation
- Hot reload
- Per-agent configuration

---

## 9. KEY FILES SUMMARY

| File | Purpose | Key Types |
|------|---------|-----------|
| `internal/temporal/workflows_dag.go` | DAG workflow | `TddDagWorkflow`, `Task`, `DAGWorkflowInput` |
| `internal/temporal/workflows_tcr.go` | TCR workflow | `TCRWorkflow`, `TCRWorkflowInput`, `TCRWorkflowResult` |
| `internal/temporal/activities_cell.go` | Cell activities | `CellActivities`, `BootstrapOutput`, `TaskOutput` |
| `internal/temporal/activities_shell.go` | Shell activities | `ShellActivities` |
| `internal/temporal/globals.go` | Global managers | `InitializeGlobals`, `GetManagers` |
| `internal/infra/ports.go` | Port management | `PortManager`, `PortManagerInterface` |
| `internal/infra/server.go` | Server lifecycle | `ServerManager`, `ServerHandle` |
| `internal/infra/worktree.go` | Git worktrees | `WorktreeManager`, `WorktreeInfo` |
| `internal/infra/interfaces.go` | Infrastructure interfaces | All *Interface types |
| `internal/workflow/activities.go` | High-level activities | `Activities`, `CellBootstrap` |
| `internal/config/config.go` | Configuration | `Config`, all Config* types |
| `pkg/agent/manager.go` | Agent tracking | `Agent`, `Manager` |
| `pkg/coordinator/coordinator.go` | Multi-agent coordination | `Coordinator`, `Status` |
| `cmd/reactor/main.go` | Main orchestrator | `Config`, `main` |

---

## 10. INTEGRATION POINTS

### 10.1 OpenCode Integration

**How it works**:
1. Reactor bootstraps OpenCode server on allocated port
2. Server runs in isolated worktree
3. Reactor sends HTTP requests with prompts
4. OpenCode executes code changes
5. Reactor captures output and file modifications

**Integration Points**:
- `ServerManager.BootServer()`: Start OpenCode
- `Activities.ExecuteTask()`: Send prompt
- `Activities.RunTests()`: Run test suite
- `Activities.CommitChanges()`: Commit via git

### 10.2 Agent Mail Integration

**How it works**:
1. Agents register with Agent Mail
2. Agents send/receive messages
3. File reservations prevent conflicts
4. Threads link related messages

**Integration Points**:
- `Coordinator.RegisterAgent()`: Register with Agent Mail
- `Coordinator.Sync()`: Fetch messages and status
- Agent Mail MCP server at http://localhost:8765/mcp

### 10.3 Beads Integration

**How it works**:
1. Tasks created in Beads
2. Agents claim tasks
3. Agents update task status
4. Beads tracks dependencies

**Integration Points**:
- `beads_create`: Create task
- `beads_update`: Update status
- `beads_close`: Complete task
- Beads MCP server via `beads-mcp` command

---

## 11. EXECUTION FLOW DIAGRAM

```
┌─────────────────────────────────────────────────────────────┐
│                    Reactor Service                          │
│                   (cmd/reactor/main.go)                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │  Initialize Infrastructure         │
        │  - PortManager                     │
        │  - ServerManager                   │
        │  - WorktreeManager                 │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │  Bootstrap Cell                    │
        │  - Allocate port                   │
        │  - Create worktree                 │
        │  - Start OpenCode server           │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │  Execute Task                      │
        │  - Send prompt to OpenCode         │
        │  - Capture output                  │
        │  - Track file modifications        │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │  Run Tests                         │
        │  - Execute test suite              │
        │  - Capture results                 │
        └────────────┬───────────────────────┘
                     │
        ┌────────────┴───────────────┐
        │                            │
        ▼                            ▼
   ┌─────────────┐          ┌──────────────┐
   │ Tests Pass  │          │ Tests Fail   │
   └──────┬──────┘          └──────┬───────┘
          │                        │
          ▼                        ▼
   ┌─────────────┐          ┌──────────────┐
   │ Commit      │          │ Revert       │
   │ Changes     │          │ Changes      │
   └──────┬──────┘          └──────┬───────┘
          │                        │
          └────────────┬───────────┘
                       │
                       ▼
        ┌────────────────────────────────────┐
        │  Teardown Cell                     │
        │  - Stop OpenCode server            │
        │  - Remove worktree                 │
        │  - Release port                    │
        └────────────────────────────────────┘
```

---

## 12. NEXT STEPS FOR OPENCODE INTEGRATION

1. **Implement OpenCode HTTP Client**
   - Wrapper for OpenCode API
   - Prompt submission
   - Response parsing
   - Error handling

2. **Enhance Task Execution**
   - Map Beads tasks to OpenCode prompts
   - Capture file modifications
   - Track test results
   - Update Beads with status

3. **Implement Agent Coordination**
   - Agent Mail messaging
   - File reservation management
   - Multi-agent task distribution

4. **Add Monitoring & Observability**
   - Metrics collection
   - Logging aggregation
   - Health checks
   - Performance tracking

5. **Extend Test Coverage**
   - Mock OpenCode server
   - Integration test fixtures
   - Performance benchmarks
   - Chaos testing scenarios
