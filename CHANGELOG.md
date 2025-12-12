# Changelog

All notable changes to the Open Swarm project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [6.1.0] - 2025-12-12

### Added

#### Temporal Integration (Complete)

- **Temporal SDK Integration**: Added Temporal v1.38.0 for workflow orchestration and distributed execution
- **Activity Layer Implementation**: Complete serializable activity wrappers for all infrastructure operations
  - `activities_shell.go`: Shell command execution using bitfield/script with proper error handling
  - `activities_cell.go`: Cell lifecycle management (bootstrap, execute, teardown) with structured logging
  - Serializable input/output types for all activities enabling Temporal's persistence layer

- **Workflow Orchestration**: Two complementary workflow patterns for different execution scenarios
  - **DAG Workflow** (`workflows_dag.go`): Directed Acyclic Graph execution with:
    - Toposort-based dependency resolution for parallel-safe task ordering
    - Parallel task execution using Temporal selectors
    - Test-Driven Development (TDD) integration with per-task test cycles
    - Deterministic execution guarantees through Temporal's event sourcing

  - **Test-Commit-Revert (TCR) Workflow** (`workflows_tcr.go`): Tight loop development pattern featuring:
    - Automated test execution with commit-only-on-success semantics
    - Saga pattern implementation for guaranteed cleanup (rollback on failure)
    - Idempotent workflow design for resilience against transient failures
    - Integration with existing Reactor infrastructure

- **Infrastructure as Code**: Temporal ecosystem setup
  - `docker-compose.yml`: Complete Temporal cluster with Temporal Server, Temporal UI, and PostgreSQL persistence
  - Pre-configured namespaces and worker queues ready for production use

- **Worker and Client Binaries**: Complete CLI tooling for Temporal integration
  - `cmd/temporal-worker/main.go`: Worker process registering all workflows and activities
    - Configured for 50 concurrent activities
    - Automatic activity heartbeating for long-running tasks
    - Proper error handling and graceful shutdown

  - `cmd/reactor-client/main.go`: CLI client for workflow submission
    - TCR workflow submission with task ID and prompt parameters
    - Workflow execution ID tracking for observability
    - Integration with existing task management

- **Example Workflows**: Reference implementations for common patterns
  - `examples/build-test-dag.json`: Multi-stage build pipeline with dependencies
  - `examples/multi-stage-dag.json`: Complex DAG with parallel branches
  - `examples/simple-tcr.json`: Minimal TCR workflow for feature development

#### Enhanced Features

- **Singleton Pattern**: Non-serializable singleton pattern in `internal/temporal/globals.go` for managing:
  - Port allocation manager
  - Server lifecycle manager
  - Git worktree management
  - These components remain in main process while Temporal handles distributed orchestration

- **GitHub Actions CI**: Automated testing and validation pipeline
  - Runs on push and pull requests
  - Tests all phases and integration points
  - Validates docker-compose configuration

- **Documentation**: Comprehensive guides for Temporal integration
  - `docs/DEPLOYMENT.md`: Production deployment strategies and scaling
  - `docs/MONITORING.md`: Observability and metrics tracking
  - `docs/TCR-WORKFLOW.md`: Test-Commit-Revert pattern guide
  - `docs/DAG-WORKFLOW.md`: Directed Acyclic Graph execution guide
  - Example workflow definitions and templates

### Changed

- **Project Structure**: Added Temporal support layer
  ```
  internal/temporal/
  ├── globals.go              # Singleton pattern for non-serializable managers
  ├── activities_cell.go      # Cell lifecycle activities
  ├── activities_shell.go     # Shell command activities
  ├── workflows_dag.go        # DAG workflow orchestration
  └── workflows_tcr.go        # Test-Commit-Revert workflow
  ```

- **Dependencies**: Added distributed execution support
  - `go.temporal.io/sdk v1.38.0`: Core workflow orchestration
  - `github.com/gammazero/toposort v0.1.1`: DAG dependency resolution
  - `github.com/bitfield/script v0.24.1`: Safe shell command execution

- **Architecture**: Reactor-SDK now supports two execution modes
  - **Local SDK Mode** (v6.0.0): Single machine orchestration via OpenCode SDK
  - **Distributed Temporal Mode** (v6.1.0): Cluster-wide execution via Temporal workflows

### Technical Details

#### Phase 1: Dependencies and Docker Compose (Dec 12, 13:02)
- Added Temporal v1.38.0, toposort v0.1.1, bitfield/script v0.24.1
- Configured docker-compose.yml with Temporal Server, UI, and PostgreSQL persistence
- Tasks closed: open-swarm-64o.6, open-swarm-64o.7, open-swarm-64o

#### Phase 2: Singleton Infrastructure Pattern (Dec 12, 13:03)
- Implemented singleton pattern in globals.go for non-serializable managers
- Documented architectural rationale for singleton vs. distributed state
- Tasks closed: open-swarm-a2v.2, open-swarm-a2v

#### Phase 3: Activity Layer (Dec 12, 13:05)
- Created activities_shell.go with bitfield/script integration
- Created activities_cell.go with complete cell lifecycle
- Implemented serializable input/output types for Temporal persistence
- Tasks closed: open-swarm-ud9.3, open-swarm-ud9.4, open-swarm-ud9

#### Phase 4: Workflow Layer (Dec 12, 13:07) - CRITICAL PATH
- Implemented workflows_dag.go with toposort-based dependency resolution
- Created workflows_tcr.go with Test-Commit-Revert pattern and saga cleanup
- Integrated parallel task execution with Temporal selectors
- Tasks closed: open-swarm-fb3.3, open-swarm-fb3.4, open-swarm-fb3

#### Phase 5: Binaries and Deployment (Dec 12, 13:09)
- Created temporal-worker/main.go with workflow/activity registration
- Created reactor-client/main.go with CLI workflow submission
- Both binaries fully functional and tested
- Tasks closed: open-swarm-e0y.3, open-swarm-e0y.4, open-swarm-e0y

### Migration Guide

From v6.0.0 to v6.1.0:

1. **Keep existing SDK-based orchestration**: No breaking changes to `cmd/open-swarm` or existing SDK APIs
2. **Opt-in to Temporal**: New workflows run alongside existing infrastructure
3. **Use Temporal for complex scenarios**: DAG workflows for parallel tasks, TCR for rapid development cycles
4. **Run both modes**: Local SDK mode continues working; Temporal adds distributed capabilities

### Installation

```bash
# Build the new binaries
go build -o bin/temporal-worker ./cmd/temporal-worker
go build -o bin/reactor-client ./cmd/reactor-client

# Start Temporal ecosystem
docker-compose up -d

# Start worker process
./bin/temporal-worker

# Submit workflows
./bin/reactor-client tcr --task TASK-001 --prompt "Your task here"
```

---

## [6.0.0] - 2025-12-12

### Added

#### Core Reactor-SDK Framework

- **Enterprise Agent Orchestration**: SDK-driven Reactor with bare-metal isolation for multiple OpenCode AI agents
- **Architecture Trinity**: Three-tier architecture combining isolation, communication, and control
  - **Brain**: `opencode serve` headless servers on unique ports (8000-9000 range)
  - **Nerve**: OpenCode Go SDK (v0.19.1) for type-safe API interaction
  - **Hand**: Git worktrees for filesystem isolation per agent

#### Infrastructure Management

- **Port Allocation Manager**: Efficient port management across 1000-port range
  - Prevents port conflicts across concurrent agents
  - Tracks allocation state with automatic release on cleanup

- **Server Lifecycle Management**: OpenCode server process orchestration
  - Automatic spawning with working directory isolation
  - Health check probes (10s timeout, 200ms intervals) ensuring server readiness
  - Graceful shutdown with process group termination (INV-005)
  - Recovery strategies for boot failures (R3: retry with exponential backoff)

- **Git Worktree Management**: Independent filesystem environments per cell
  - Automatic worktree creation on main branch
  - Safe cleanup with `git worktree prune`
  - Parallel operation without file contention

#### Architectural Invariants

Six immutable invariants enforced by the architecture (from Tessl spec):

1. **INV-001**: Each agent runs `opencode serve` on unique port via Port Manager
2. **INV-002**: Server working directory set to Git Worktree for isolation
3. **INV-003**: Supervisor waits for healthcheck (200 OK) before SDK connection
4. **INV-004**: SDK Client configured with specific BaseURL (localhost:PORT)
5. **INV-005**: Server process killed on workflow activity completion
6. **INV-006**: Command execution uses SDK `client.Command.Execute` only

#### Execution Patterns

- **Test-Commit-Revert (TCR) Pattern**: Tight-loop development with guaranteed safety
  - Execute user prompt via SDK
  - Retrieve modified files from execution context
  - Run tests via SDK shell command
  - Commit on pass, reset --hard on failure
  - Atomic operations ensuring code always compiles

- **Single Task Execution**: Bootstrap → Execute → Test → Commit/Revert → Teardown
- **Parallel Mode**: N isolated cells executing independent tasks simultaneously
  - Default: 50 concurrent agent limit
  - Port range: 1000 available slots
  - System resources: CPU/memory per cell

#### Developer Experience

- **Type-Safe SDK Integration**: OpenCode Go SDK v0.19.1 with complete API coverage
- **Command Execution**: `client.Command.Execute` for arbitrary shell commands
- **File Operations**: `client.File.Status` for modification tracking
- **Session Management**: Stateful prompt execution via `client.Session.Prompt`
- **Full Observability**: Request/response inspection, structured logging

#### Configuration & Deployment

- **Port Range**: Configurable 8000-9000 (1000 ports, supports up to 50 concurrent agents)
- **Healthcheck Settings**: 10s timeout, 200ms interval polling
- **Shutdown Timeout**: 5s graceful period before force kill
- **Resource Limits**: Support for cgroups/Docker limits per cell (estimated 2GB RAM, 1-2 cores per agent)

#### Recovery & Resilience

Recovery strategies from Tessl specification:
- **R3**: Retry up to 3 times with exponential backoff on server boot failure
- **RB**: Rollback (git reset --hard) on test failure
- **IG**: Ignore and warn for non-critical errors

#### Testing & Quality

- **Comprehensive Infrastructure Tests**: Port allocation, server lifecycle, worktree management
- **Test Coverage**: All core components covered with unit tests
- **Integration Tests**: Full end-to-end workflows tested
- **Binary Compilation**: 9.8MB optimized binary ready for production

#### Documentation

- `REACTOR.md`: Complete architecture and usage guide
- `README.md`: Multi-agent coordination framework overview
- `QUICKSTART.md`: 10-minute setup and first session guide
- `AGENTS.md`: Comprehensive agent instructions and patterns
- Example workflows and deployment guides

#### Project Structure

```
open-swarm/
├── cmd/
│   └── open-swarm/        # CLI application (future)
├── internal/
│   ├── infra/
│   │   ├── ports.go       # Port allocation (INV-001)
│   │   ├── server.go      # Server lifecycle (INV-002, INV-003, INV-005)
│   │   └── worktree.go    # Git worktree management
│   ├── agent/
│   │   ├── client.go      # SDK wrapper (INV-004, INV-006)
│   │   └── types.go       # Data structures
│   └── workflow/
│       ├── activities.go  # Workflow activities
│       └── tcr_workflow.go # TCR pattern implementation
├── docker-compose.yml     # Containerized Temporal setup
├── go.mod                 # Dependencies
└── documentation files    # Complete guides
```

### Dependencies

- **OpenCode SDK**: v0.19.1 (provider-agnostic AI coding agent platform)
- **Go**: 1.25+
- **Architecture Support**: Linux/macOS/Windows via Go's cross-platform capabilities

### Verified Functionality

- ✅ Port allocation manager with conflict prevention
- ✅ Server lifecycle management with health checks
- ✅ Git worktree isolation and cleanup
- ✅ OpenCode SDK integration and command execution
- ✅ Test-Commit-Revert pattern implementation
- ✅ All 6 architectural invariants enforced
- ✅ Infrastructure tests passing
- ✅ Binary compiles and runs successfully

### Known Limitations

- Single-machine orchestration (horizontal scaling requires external coordination)
- Token/cost visibility limited by SDK abstraction (time-boxing as cost control)
- Process group management OS-dependent (currently optimized for Unix-like systems)

### Future Work

- v6.1.0: Full go-workflows integration for complex DAGs
- v6.1.0: Temporal distributed execution with workflows
- v6.2.0: Metrics export in Prometheus format
- v6.2.0: Distributed mode with message queue support
- v6.2.0: Web UI for monitoring cells
- v7.0.0: Kubernetes operator for cluster deployment
- v7.0.0: Auto-scaling based on queue depth
- v7.0.0: Multi-region support

---

[Unreleased]: https://github.com/yourusername/open-swarm/compare/v6.1.0...HEAD
[6.1.0]: https://github.com/yourusername/open-swarm/compare/v6.0.0...v6.1.0
[6.0.0]: https://github.com/yourusername/open-swarm/releases/tag/v6.0.0
