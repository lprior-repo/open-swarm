# Monitoring Guide

This guide explains how to monitor the Open Swarm system including Temporal workflows, agent cells, port usage, and system logs.

## Table of Contents

1. [Temporal Web UI](#temporal-web-ui)
2. [Workflow History Interpretation](#workflow-history-interpretation)
3. [Agent Cell Tracking](#agent-cell-tracking)
4. [Port Usage Monitoring](#port-usage-monitoring)
5. [Viewing Logs](#viewing-logs)
6. [Metrics and Performance](#metrics-and-performance)
7. [Troubleshooting](#troubleshooting)

## Temporal Web UI

### Accessing the Web UI

The Temporal Web UI is automatically available when running the Docker Compose stack.

#### Start Services
```bash
make docker-up
```

#### Access the UI
- **URL:** http://localhost:8233
- **Temporal Server:** localhost:7233
- **PostgreSQL:** localhost:5432

### Web UI Overview

The Temporal Web UI provides several key views:

#### Workflows View
- **Main Dashboard:** Shows all running and completed workflows
- **Task Queue:** `reactor-task-queue` - the default task queue for Open Swarm workflows
- **Status Indicators:**
  - `Green`: Running workflows
  - `Blue`: Completed workflows
  - `Orange`: Failed workflows
  - `Red`: Terminated workflows

#### Key Metrics in Dashboard
- **Workflow Execution Count:** Total number of workflow executions
- **Failed Executions:** Number of failed workflows
- **Timeout Executions:** Workflows that timed out
- **Average Execution Time:** Performance metric

### Navigating the Web UI

1. **Search Workflows**
   - Filter by workflow name, status, or task queue
   - Example: Search for `TCRWorkflow` to see all Test-Commit-Revert executions

2. **View Workflow Details**
   - Click on any workflow execution to see:
     - Current state and status
     - Input parameters
     - Timeline of activities
     - Results and errors

3. **Task Queue Monitor**
   - Navigate to "Task Queues" tab
   - Monitor `reactor-task-queue`:
     - Backlog size (pending activities)
     - Number of workers connected
     - Worker heartbeat status

## Workflow History Interpretation

### Understanding Workflow Events

Each workflow maintains a detailed history of all events. Open Swarm uses two main workflow types:

#### TCR Workflow (Test-Commit-Revert)
Located in `/home/lewis/src/open-swarm/internal/temporal/workflows_tcr.go`

Sequence of activities:
1. **BootstrapCell** - Initialize isolated agent environment
2. **ExecuteTask** - Run the AI agent prompt
3. **RunTests** - Execute test suite
4. **CommitChanges** (success) OR **RevertChanges** (failure) - Persist or discard
5. **TeardownCell** - Clean up resources

Example workflow history event:
```
[TCRWorkflow] Starting TCR Workflow taskID=task-123
  → BootstrapCell (cellID=cell-001, port=8000)
  → ExecuteTask (prompt="Implement feature X")
  → RunTests (success=true)
  → CommitChanges (message="Feature X implemented")
  → TeardownCell (cellID=cell-001)
Result: Success
```

#### TDD DAG Workflow (Test-Driven Development)
Located in `/home/lewis/src/open-swarm/internal/temporal/workflow_dag.go`

More complex workflow with:
- Multiple parallel activities
- Dependency tracking
- Intermediate state persistence

### Reading Event Timeline

In the Web UI workflow details:

1. **Event Type:** Look for activity execution events
   - `ActivityTaskStarted` - Activity began
   - `ActivityTaskCompleted` - Activity finished successfully
   - `ActivityTaskFailed` - Activity encountered an error
   - `WorkflowExecutionCompleted` - Entire workflow finished

2. **Duration:** Shows how long each activity took
   - Quick activities (< 10s): Usually infrastructure setup
   - Long activities (> 1min): Agent processing or tests running

3. **Heartbeat Messages:** Intermediate progress updates
   - "allocating resources" - Port and worktree allocation
   - "executing prompt" - AI agent is working
   - Messages appear in activity details

### Identifying Issues

**Failed Activities:**
```
ActivityTaskFailed:
  Reason: "Activity execution failed"
  Details: "no available ports in range 8000-9000"
  Suggestion: Too many concurrent agents; release cell resources
```

**Timeout Issues:**
```
ActivityTaskTimedOut:
  Reason: "StartToCloseTimeout exceeded"
  Duration: 10m 5s (limit was 10m)
  Suggestion: Increase timeout or investigate slow agent/test execution
```

**Revert Events:**
```
ExecuteTask completed → RunTests failed → RevertChanges → TeardownCell
Result: Success (revert completed) but changes were discarded
```

## Agent Cell Tracking

### What is an Agent Cell?

An agent cell is an isolated environment where one AI agent works:
- Unique port allocation (8000-9000 range)
- Git worktree (isolated copy of the repository)
- OpenCode server instance
- Complete execution context

### Monitoring Active Cells

#### Via Temporal Web UI

1. Go to "Workflows" tab
2. Filter by workflow name: `TCRWorkflow` or `TddDagWorkflow`
3. Look at "Status" column:
   - "RUNNING" = Active cell currently executing
   - "COMPLETED" = Cell finished and torn down
   - "FAILED" = Cell encountered error

#### Via Command Line

**Check Temporal CLI:**
```bash
# List recent workflows
tctl workflow list --pagesize 10

# Describe specific workflow
tctl workflow describe -w <workflow-id>

# List all activities in workflow
tctl workflow showhistory -w <workflow-id>
```

**Check Active Processes:**
```bash
# See all opencode server instances
ps aux | grep "opencode serve"

# Output example:
# user  8000  opencode serve --port 8000 --hostname localhost
# user  8001  opencode serve --port 8001 --hostname localhost
```

**Check Git Worktrees:**
```bash
# List active worktrees
git worktree list

# Output:
# /path/to/repo                      hash [main]
# /path/to/repo/worktrees/cell-001  hash [cell-001]
# /path/to/repo/worktrees/cell-002  hash [cell-002]
```

### Cell Lifecycle Tracking

**Cell Input Parameters:**
```
BootstrapInput {
  CellID: "cell-abc123"
  Branch: "feature/new-feature"
}

BootstrapOutput {
  CellID:       "cell-abc123"
  Port:         8000
  WorktreeID:   "cell-abc123"
  WorktreePath: "/repo/worktrees/cell-abc123"
  BaseURL:      "http://localhost:8000"
  ServerPID:    12345
}
```

**Cell Status Tracking:**
```
┌─────────────┬──────────┬────────────┬─────────────────┐
│ CellID      │ Port     │ Status     │ Elapsed Time    │
├─────────────┼──────────┼────────────┼─────────────────┤
│ cell-001    │ 8000     │ RUNNING    │ 2m 15s          │
│ cell-002    │ 8001     │ COMPLETED  │ 3m 47s          │
│ cell-003    │ 8002     │ FAILED     │ 45s (error)     │
└─────────────┴──────────┴────────────┴─────────────────┘
```

### Resource Allocation per Cell

Each cell allocates:
- **Port:** 1 unique port from 8000-9000 range
- **Worktree:** ~500MB-1GB (depends on repository size)
- **Memory:** ~200-500MB (for opencode server + agent)
- **CPU:** Variable (during task execution)

Monitor via worker initialization in code:
```go
// From cmd/temporal-worker/main.go
temporal.InitializeGlobals(
  8000,      // portMin
  9000,      // portMax (supports up to 1000 cells)
  ".",       // repoDir
  "./worktrees",  // worktreeBase
)
```

## Port Usage Monitoring

### Port Allocation Strategy

Open Swarm allocates ports in the range **8000-9000** for agent cells.

- **Total available ports:** 1001 (8000-9000 inclusive)
- **Worker configuration:** MaxConcurrentActivityExecutionSize = 50
- **Scalability:** Supports up to 1000 concurrent cells (theoretical limit)

### Monitoring Port Allocation

#### Check Allocated Ports

**Using netstat:**
```bash
# List all ports in 8000-9000 range
netstat -tlnp | grep -E '8[0-9]{3}'

# Output example:
# LISTEN   user  opencode  8000
# LISTEN   user  opencode  8001
# LISTEN   user  opencode  8002
```

**Using lsof:**
```bash
# List all processes using ports 8000-9000
lsof -i :8000-9000

# More detailed:
lsof -i '@localhost' | grep -E '8[0-9]{3}'
```

**Using ss (modern alternative):**
```bash
# Show socket statistics for range
ss -tlnp 'sport >= :8000 and sport <= :9000'
```

### Port Manager Metrics

The PortManager (in `internal/infra/ports.go`) provides:
- `AllocatedCount()` - Number of currently allocated ports
- `AvailableCount()` - Number of free ports
- `IsAllocated(port)` - Check if specific port is in use

These are accessed within workflows:
```go
pm, _, _ := temporal.GetManagers()
allocated := pm.AllocatedCount()  // Currently in use
available := pm.AvailableCount()  // Free ports
```

### Port Exhaustion Scenarios

**Indicator:** Worker cannot allocate new cells
```
Error: "no available ports in range 8000-9000 (all 1001 ports allocated)"
```

**Causes:**
1. Teardown activities failed silently
2. OpenCode server processes hung
3. Cells not properly released after failure

**Resolution:**
```bash
# 1. Check for stuck processes
ps aux | grep "opencode serve" | grep -v grep

# 2. Kill stuck servers if needed
pkill -f "opencode serve"

# 3. Verify ports are released
netstat -tlnp | grep -E '8[0-9]{3}' | wc -l

# 4. Restart worker
make docker-down
make docker-up
make run-worker
```

### Monitoring During Load

For monitoring concurrent cells:

```bash
# Watch port allocation in real-time
watch -n 1 'netstat -tlnp 2>/dev/null | grep -E "8[0-9]{3}" | wc -l'

# Or check worktrees
watch -n 1 'git worktree list | wc -l'
```

## Viewing Logs

### Worker Logs

The Temporal worker logs activity execution and lifecycle events.

#### Start Worker with Log Output
```bash
# Build and run worker
make run-worker

# Output includes:
# 🚀 Reactor-SDK Temporal Worker v6.1.0
# 🔧 Initializing global managers...
# ✅ Connected to Temporal server
# 📋 Registered workflows and activities
# ⚙️  Worker listening on task queue: reactor-task-queue
```

#### Filtering Worker Logs

**By Activity Type:**
```bash
# Grep for bootstrap activities
make run-worker 2>&1 | grep -i bootstrap

# Grep for execute activities
make run-worker 2>&1 | grep -i execute

# Grep for teardown activities
make run-worker 2>&1 | grep -i teardown
```

**By Error Level:**
```bash
# Show only errors
make run-worker 2>&1 | grep -i "error\|failed"

# Show errors and warnings
make run-worker 2>&1 | grep -iE "error|failed|warning"
```

### Activity Logs

Activities log via the Temporal SDK logger:

```go
// From activities_cell.go
logger := activity.GetLogger(ctx)
logger.Info("Bootstrapping cell", "cellID", input.CellID)
logger.Info("Executing task", "taskID", task.TaskID)

// Heartbeat messages
activity.RecordHeartbeat(ctx, "allocating resources")
activity.RecordHeartbeat(ctx, "executing prompt")
```

#### Access Activity Logs in Web UI

1. Open workflow execution in Temporal Web UI
2. Click on activity (e.g., "BootstrapCell")
3. View "Details" panel showing:
   - Activity state
   - Start/end times
   - Heartbeat messages
   - Result or error

### Docker Service Logs

#### Temporal Server Logs
```bash
# View temporal container logs
make docker-logs

# Or directly
docker-compose logs temporal -f

# Search for errors
docker-compose logs temporal | grep -i error
```

#### PostgreSQL Logs
```bash
# View database logs
docker-compose logs postgresql -f

# Check for connection errors
docker-compose logs postgresql | grep -i "error\|refused"
```

#### Combined Service Health
```bash
# Check all service health
docker-compose ps

# Output:
# NAME            STATUS
# postgresql      Up 5 minutes (healthy)
# temporal        Up 3 minutes (healthy)
```

### OpenCode Server Logs

Each agent cell runs an OpenCode server. Logs are generated in the worktree:

```bash
# Find opencode logs in worktree
find ./worktrees -name "*.log" -type f

# Tail logs from latest activity
ls -lt ./worktrees/*/opencode.log | head -1 | awk '{print $NF}' | xargs tail -f
```

### Workflow Execution Logs

Access via Temporal Web UI for detailed workflow context:

1. **Workflow Details Page:**
   - Click workflow execution ID
   - "Details" tab shows timing
   - "History" tab shows all events chronologically

2. **Activity Details:**
   - Click individual activity
   - View logs recorded during execution
   - Check heartbeat progress

3. **Error Details:**
   - Failed activities show error stack traces
   - Timeout details show duration exceeded
   - Failure reasons documented

## Metrics and Performance

### Temporal Metrics

Temporal server exposes metrics on port `8233`. While Prometheus integration is not currently configured in Open Swarm, the foundation is ready.

#### Key Metrics to Monitor

**Workflow Metrics:**
- `workflow_execution_started_total` - Total workflows started
- `workflow_execution_completed_total` - Total completed workflows
- `workflow_execution_failed_total` - Total failed workflows
- `workflow_execution_duration_seconds` - Duration histogram

**Activity Metrics:**
- `activity_execution_started_total` - Total activity executions
- `activity_execution_completed_total` - Completed activities
- `activity_execution_failed_total` - Failed activities
- `activity_execution_duration_seconds` - Duration histogram

**Worker Metrics:**
- `worker_task_slots_available` - Available concurrency slots
- `worker_task_slots_used` - Currently used slots

### Performance Baselines

#### Expected Timing

**TCR Workflow Phases:**
```
BootstrapCell:    5-15 seconds (port allocation, worktree creation)
ExecuteTask:      30 seconds - 10 minutes (depends on prompt complexity)
RunTests:         10-60 seconds (test suite execution)
CommitChanges:    5 seconds (git operations)
TeardownCell:     5-10 seconds (cleanup)
─────────────────────────────────────
Total:            1-15 minutes per workflow
```

**Activity Concurrency:**
```
MaxConcurrentActivityExecutionSize: 50
MaxConcurrentWorkflowTaskExecutionSize: 10
MaxConcurrentLocalActivityExecutionSize: 100
```

This allows:
- **50 parallel agent cells** (ExecuteTask activities)
- **10 simultaneous workflow decisions**
- **100 local activities** (non-blocking tasks)

### Memory and CPU Usage

**Per Cell:**
- OpenCode server: ~200-300MB
- Git worktree: ~500MB-1GB
- Temporal SDK overhead: ~50MB

**Total for 50 concurrent cells:**
- Memory: ~50GB (manageable on modern servers)
- CPU: Variable (depends on agent work intensity)

**Example Host Requirements:**
```
For 50 concurrent cells:
- RAM: 64GB+ recommended
- CPU: 16 cores minimum
- Disk: 100GB+ (for worktrees)
- Network: 100Mbps+ for stability
```

### Workflow Duration Monitoring

**In Temporal Web UI:**

1. View workflow execution list
2. "Duration" column shows total workflow time
3. Click workflow to see phase breakdown

**Command Line:**
```bash
# Get workflow duration via CLI
tctl workflow describe -w <workflow-id> --raw
```

## Troubleshooting

### Common Issues and Solutions

#### Issue: Worker Cannot Connect to Temporal

**Symptoms:**
```
❌ Unable to create Temporal client: connection refused
```

**Solutions:**
```bash
# 1. Start Docker services
make docker-up

# 2. Wait for services to be healthy
sleep 15

# 3. Verify Temporal server is running
docker-compose ps temporal

# 4. Check Temporal connectivity
curl http://localhost:7233/health 2>&1

# 5. Restart services
make docker-down
make docker-up
sleep 20
```

#### Issue: All Ports Exhausted

**Symptoms:**
```
Error: "no available ports in range 8000-9000 (all 1001 ports allocated)"
Cells not starting
```

**Investigation:**
```bash
# Check active processes
ps aux | grep "opencode serve" | wc -l

# Check open ports
netstat -tlnp | grep -E '8[0-9]{3}' | wc -l

# Check git worktrees
git worktree list | wc -l
```

**Solutions:**
```bash
# 1. Kill orphaned servers (careful!)
pkill -f "opencode serve"

# 2. Clean up git worktrees
git worktree prune

# 3. Hard reset if necessary
make clean
git worktree list --porcelain | awk '{print $2}' | xargs -I {} rm -rf {}

# 4. Restart everything
make docker-down
make docker-up
sleep 20
make run-worker
```

#### Issue: Workflow Timeout

**Symptoms:**
```
ActivityTaskTimedOut: StartToCloseTimeout exceeded (10m limit)
```

**Investigation:**
```bash
# Check how long ExecuteTask took
# In Temporal Web UI: Click activity, check "Started" and "Closed" times

# Check agent logs
tail -100 ./worktrees/*/opencode.log
```

**Solutions:**
```bash
# Option 1: Increase timeout in workflow
// In workflows_tcr.go
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 20 * time.Minute,  // Increase from 10
}

# Option 2: Optimize prompt execution
# - Smaller prompts = faster execution
# - Ensure tests run quickly
# - Monitor agent resource usage

# Option 3: Use TDD DAG workflow for parallelization
```

#### Issue: Cell Resource Leaks

**Symptoms:**
```
Worktree space grows without bound
Free disk space decreasing
```

**Investigation:**
```bash
# Check worktree count
git worktree list | wc -l

# Check disk usage
du -sh ./worktrees

# Look for incomplete teardowns
git worktree list | grep "detached"
```

**Solutions:**
```bash
# 1. Clean up orphaned worktrees
git worktree prune

# 2. Manually remove stuck worktrees
rm -rf ./worktrees/cell-*

# 3. Monitor teardown activities
# Watch Temporal Web UI for TeardownCell events
# Should see one for each BootstrapCell

# 4. Add monitoring script
# (See next section)
```

### Monitoring Scripts

#### Script: Monitor Port Usage

```bash
#!/bin/bash
# monitor-ports.sh

echo "Monitoring port usage..."
while true; do
  COUNT=$(netstat -tlnp 2>/dev/null | grep -E '8[0-9]{3}' | wc -l)
  TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
  echo "[$TIMESTAMP] Ports in use: $COUNT/1001"
  sleep 10
done
```

#### Script: Monitor Cell Lifecycle

```bash
#!/bin/bash
# monitor-cells.sh

echo "Monitoring active cells..."
while true; do
  CELLS=$(git worktree list | grep -c "cell-")
  PROCESSES=$(ps aux | grep -c "opencode serve")
  TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
  echo "[$TIMESTAMP] Cells: $CELLS | Processes: $PROCESSES"
  sleep 10
done
```

#### Script: Cleanup Stale Resources

```bash
#!/bin/bash
# cleanup-stale.sh

echo "Cleaning up stale resources..."

# 1. Prune orphaned worktrees
git worktree prune
echo "✓ Pruned orphaned worktrees"

# 2. Kill opencode servers older than 1 hour
find /proc -name "opencode" -mmin +60 -exec kill {} \;
echo "✓ Killed stale opencode processes"

# 3. Clean temporary files
rm -rf ./worktrees/*/tmp/*
echo "✓ Cleaned temporary files"

echo "Cleanup complete"
```

### Health Check Commands

```bash
#!/bin/bash
# health-check.sh

echo "=== Open Swarm Health Check ==="

# Check Docker services
echo -n "Temporal Server: "
docker-compose ps temporal | grep -q "Up" && echo "✓" || echo "✗"

echo -n "PostgreSQL: "
docker-compose ps postgresql | grep -q "Up" && echo "✓" || echo "✗"

# Check Temporal connectivity
echo -n "Temporal Connectivity: "
curl -s http://localhost:7233/health > /dev/null && echo "✓" || echo "✗"

# Check Web UI
echo -n "Web UI (8233): "
curl -s http://localhost:8233 > /dev/null && echo "✓" || echo "✗"

# Check active ports
echo -n "Active Cells: "
ps aux | grep -c "opencode serve" | xargs echo

# Check worktrees
echo -n "Worktree Count: "
git worktree list | grep -c "cell-" | xargs echo

# Check available ports
echo -n "Available Ports: "
USED=$(netstat -tlnp 2>/dev/null | grep -E '8[0-9]{3}' | wc -l)
echo "$((1001 - USED))/1001"

echo ""
echo "✓ Health check complete"
```

## Summary

Open Swarm provides comprehensive monitoring capabilities:

| Component | Tool | Location | Key Metric |
|-----------|------|----------|-----------|
| Workflows | Temporal Web UI | http://localhost:8233 | Execution status, duration |
| Activities | Web UI Details | Activity timeline | Start/end time, heartbeats |
| Cells | Temporal + PS | Workflows + processes | Port allocation, status |
| Ports | netstat/lsof | System | Allocated/available count |
| Logs | Docker/stdout | Containers + stdout | Error messages, warnings |
| Performance | Task queue monitor | Web UI | Backlog, worker count |

For production deployment, consider:
1. Enabling Prometheus metrics collection
2. Setting up Grafana dashboards
3. Configuring alerting on workflow failures
4. Implementing structured logging aggregation
