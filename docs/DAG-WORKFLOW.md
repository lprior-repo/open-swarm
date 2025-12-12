# DAG Workflow Usage Guide

**Version:** 1.0
**Architecture:** Temporal-based DAG Execution with Parallel Task Scheduling
**Focus:** Dependency-Driven Workflows with Test-Driven Development Loop

## Overview

The DAG (Directed Acyclic Graph) Workflow is a sophisticated task orchestration system built on Temporal.io that enables:

- **Dependency Resolution** - Automatically sequence tasks based on their dependencies
- **Parallel Execution** - Run independent tasks simultaneously for optimal performance
- **TDD Loop Integration** - Continuously retry failed workflows until success with human intervention signals
- **Signal Handling** - Wait for external signals (e.g., "FixApplied") to resume execution
- **Topological Sorting** - Guarantee correct execution order even with complex dependency chains

### When to Use DAG Workflows

Use DAG workflows when you need to:

1. **Execute multiple interdependent tasks** - Tasks that depend on other tasks' completion
2. **Optimize parallel execution** - Run independent tasks simultaneously
3. **Implement CI/CD pipelines** - Multiple build, test, and deployment stages
4. **Test-driven development cycles** - Automatically retry until all tasks pass
5. **Handle failure recovery** - Pause and resume via human signals

### When to Use TCR Workflows Instead

Use Test-Commit-Revert (TCR) workflows when you need to:

- Execute a single isolated task in an OpenCode cell
- Guarantee atomicity (all-or-nothing execution)
- Avoid complexity of dependency management

## DAG Resolution

### Understanding the DAG Model

A DAG consists of **Tasks** and **Dependencies**:

```
Task {
  Name:    "build"
  Command: "go build ./..."
  Deps:    []              // No dependencies
}

Task {
  Name:    "test"
  Command: "go test ./..."
  Deps:    ["build"]       // Depends on "build"
}

Task {
  Name:    "deploy"
  Command: "kubectl apply -f manifests/"
  Deps:    ["test"]        // Depends on "test"
}
```

### Topological Sorting Algorithm

The DAG workflow uses the `toposort` library to compute the correct execution order:

```go
// Build edges: [dependency, dependent]
edges := []toposort.Edge{
  {"build", "test"},      // "test" depends on "build"
  {"test", "deploy"},     // "deploy" depends on "test"
}

// Perform topological sort
sortedNodes, err := toposort.Toposort(edges)
// Result: ["build", "test", "deploy"]
```

### Cycle Detection

If your DAG contains a cycle (circular dependency), the workflow immediately fails:

```go
if err != nil {
  return fmt.Errorf("cycle detected in DAG: %w", err)
}
```

Example of a cycle (INVALID):

```
Task A → depends on Task B
Task B → depends on Task C
Task C → depends on Task A  ❌ CYCLE!
```

### Input Structure

```go
type Task struct {
  Name    string   // Unique task identifier
  Command string   // Shell command to execute
  Deps    []string // Names of dependency tasks
}

type DAGWorkflowInput struct {
  WorkflowID string // Unique workflow identifier
  Branch     string // Git branch to work on
  Tasks      []Task // All tasks in the DAG
}
```

## TDD Loop

### The Test-Driven Development Cycle

The DAG workflow implements a retry loop that keeps running until success:

```
1. Run the DAG (execute all tasks in topological order)
   ↓
2. If all tasks succeed → Workflow completes ✅
   ↓
3. If any task fails → Human intervention needed
   ↓
4. Wait for "FixApplied" signal
   ↓
5. Increment attempt counter
   ↓
6. Repeat from step 1
```

### Code Flow

```go
attempt := 1
for {
  logger.Info("TDD Cycle Start", "attempt", attempt)

  // Run the entire DAG
  err := runDag(ctx, input.Tasks)

  if err == nil {
    logger.Info("TDD Cycle Succeeded!", "attempts", attempt)
    return nil  // ✅ Done!
  }

  // Handle failure
  logger.Error("TDD Cycle Failed", "attempt", attempt, "error", err)
  logger.Info("Waiting for 'FixApplied' signal to retry...")

  // Block until signal arrives
  var signalVal string
  signalChan := workflow.GetSignalChannel(ctx, "FixApplied")
  signalChan.Receive(ctx, &signalVal)

  logger.Info("Received FixApplied signal", "message", signalVal)

  // Restart with new attempt number
  attempt++
}
```

### What Happens When a Task Fails

1. **Failure Tracking** - The failed task name is added to `failedTasks` list
2. **Pending Tasks Complete** - Other tasks that were running continue to completion
3. **DAG Aborts** - Once all pending tasks complete, the DAG returns an error
4. **Workflow Pauses** - The TDD loop pauses and waits for the "FixApplied" signal
5. **Human Intervention** - Engineer fixes the issue in the code
6. **Signal Sent** - Send "FixApplied" signal via Temporal client
7. **DAG Retries** - The entire DAG runs again from the beginning

### Signaling from Temporal Client

To signal a running DAG workflow from your client:

```go
// Send FixApplied signal to the workflow
err := client.SignalWorkflow(
  ctx,
  "workflow-id-123",          // Your DAG workflow ID
  "run-id-123",               // Leave empty for latest
  "FixApplied",               // Signal name
  "Fix applied - retrying",   // Message
)
```

## Parallel Execution

### How Parallelism Works

The DAG execution loop launches all **runnable tasks** simultaneously:

1. **Check Dependencies** - For each task, verify all its dependencies are completed
2. **Launch Parallel Tasks** - Start all runnable tasks without waiting
3. **Track Futures** - Store workflow Futures for each running task
4. **Wait for Completion** - Use a selector to wait for the next task to finish
5. **Process Result** - Mark task as completed or failed
6. **Repeat** - Go back to step 1

### Example: Parallel Execution

Given this DAG:

```
     ┌─── build ───┐
     │             │
 main ┤─── test  ──┤─── deploy
     │             │
     └─── lint  ───┘
```

Execution timeline:

```
Iteration 1: Launch [build, test, lint] in parallel ✓
Iteration 2: Any of them finishes → Continue
Iteration 3: When all 3 done → Launch [deploy]
Iteration 4: deploy finishes → DAG complete
```

### Code Implementation

```go
// Track which tasks are running
pendingFutures := make(map[string]workflow.Future)
completed := make(map[string]bool)

for len(completed) < len(tasks) {
  // Check what can run now
  for _, taskName := range flatOrder {
    if completed[taskName] || pendingFutures[taskName] != nil {
      continue
    }

    // Check dependencies
    canRun := true
    for _, dep := range taskMap[taskName].Deps {
      if !completed[dep] {
        canRun = false
        break
      }
    }

    // Launch if ready
    if canRun {
      logger.Info("Starting task", "name", taskName)
      f := workflow.ExecuteActivity(ctx, shellActivities.RunScript, cmd)
      pendingFutures[taskName] = f  // Track it
    }
  }

  // Wait for next completion
  selector := workflow.NewSelector(ctx)

  for name := range pendingFutures {
    taskName := name
    taskFuture := pendingFutures[taskName]

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

  selector.Select(ctx)  // Block until something completes
}
```

### Limitations

- **Parallel Limit** - Temporal has internal limits on concurrent activities
- **Resource Usage** - Each running task consumes resources
- **Failure Propagation** - One failure stops the DAG immediately

## Signal Handling

### The Signal Channel

The DAG workflow listens on a named signal channel:

```go
signalChan := workflow.GetSignalChannel(ctx, "FixApplied")
signalChan.Receive(ctx, &signalVal)  // Blocks until signal arrives
```

### Signal Types

**FixApplied Signal**

Sent when a fix has been applied and the DAG should retry:

```go
// Temporal client (your CLI)
client.SignalWorkflow(
  ctx,
  "workflow-id-123",
  "",
  "FixApplied",
  "Applied fix to main.go line 42",  // Message
)
```

### Multi-Attempt Tracking

Each retry increments the attempt counter:

```
Attempt 1: Initial run → build fails
           Signal received → retry
Attempt 2: Build passes → test fails
           Signal received → retry
Attempt 3: All tasks pass ✅
```

The logger shows which attempt you're on:

```
TDD Cycle Start attempt=1
Task failed attempt=1 error=build: compilation error
Received FixApplied signal message=Fixed syntax error
TDD Cycle Start attempt=2
Task failed attempt=2 error=test: 3 tests failed
...
TDD Cycle Succeeded attempts=3
```

## Examples

### Example 1: Simple Build Pipeline

A basic three-stage pipeline: build → test → deploy.

```go
dagInput := DAGWorkflowInput{
  WorkflowID: "build-pipeline-001",
  Branch:     "main",
  Tasks: []Task{
    {
      Name:    "build",
      Command: "go build -o bin/app ./cmd/app",
      Deps:    [],  // No dependencies
    },
    {
      Name:    "test",
      Command: "go test ./...",
      Deps:    []string{"build"},  // Depends on build
    },
    {
      Name:    "deploy",
      Command: "kubectl apply -f k8s/",
      Deps:    []string{"test"},   // Depends on test
    },
  },
}
```

Execution order: build → test → deploy

### Example 2: Diamond Dependency Graph

Multiple tasks depend on a common prerequisite.

```go
dagInput := DAGWorkflowInput{
  WorkflowID: "diamond-dag-001",
  Branch:     "main",
  Tasks: []Task{
    {
      Name:    "setup",
      Command: "make setup",
      Deps:    [],  // Entry point
    },
    {
      Name:    "unit-test",
      Command: "go test ./... -unit",
      Deps:    []string{"setup"},
    },
    {
      Name:    "integration-test",
      Command: "go test ./... -integration",
      Deps:    []string{"setup"},
    },
    {
      Name:    "lint",
      Command: "golangci-lint run",
      Deps:    []string{"setup"},
    },
    {
      Name:    "coverage",
      Command: "go test ./... -cover",
      Deps:    []string{"unit-test", "integration-test", "lint"},
    },
  },
}
```

DAG structure:

```
     ┌─ unit-test ─────┐
     │                 │
setup┤─ integration-test┤─ coverage
     │                 │
     └─ lint ──────────┘
```

Execution:
1. Run setup (no dependencies)
2. In parallel: unit-test, integration-test, lint (all depend on setup)
3. When all 3 complete, run coverage (depends on all three)

### Example 3: Complex Microservices Build

Building multiple services with shared dependencies.

```go
dagInput := DAGWorkflowInput{
  WorkflowID: "microservices-build",
  Branch:     "develop",
  Tasks: []Task{
    // Shared dependencies
    {
      Name:    "proto-compile",
      Command: "protoc -I. --go_out=. proto/**/*.proto",
      Deps:    [],
    },
    {
      Name:    "deps",
      Command: "go mod download",
      Deps:    []string{"proto-compile"},
    },
    // Service A
    {
      Name:    "svc-auth-build",
      Command: "go build -o bin/auth-svc ./cmd/auth",
      Deps:    []string{"deps"},
    },
    {
      Name:    "svc-auth-test",
      Command: "go test ./cmd/auth/...",
      Deps:    []string{"svc-auth-build"},
    },
    // Service B
    {
      Name:    "svc-api-build",
      Command: "go build -o bin/api-svc ./cmd/api",
      Deps:    []string{"deps"},
    },
    {
      Name:    "svc-api-test",
      Command: "go test ./cmd/api/...",
      Deps:    []string{"svc-api-build"},
    },
    // Integration tests (need both services)
    {
      Name:    "integration-test",
      Command: "go test ./tests/integration/...",
      Deps:    []string{"svc-auth-test", "svc-api-test"},
    },
    // Final deployment
    {
      Name:    "docker-build",
      Command: "docker-compose build",
      Deps:    []string{"integration-test"},
    },
  ],
}
```

DAG structure:

```
                    ┌─ svc-auth-build ─ svc-auth-test ─┐
                    │                                   │
proto-compile ─ deps┤                                   ├─ integration-test ─ docker-build
                    │                                   │
                    └─ svc-api-build ─ svc-api-test ──┘
```

Execution timeline:
1. proto-compile (sequential start)
2. deps (after proto-compile)
3. **In parallel**: svc-auth-build, svc-api-build (both after deps)
4. **In parallel**: svc-auth-test, svc-api-test (each after their respective builds)
5. integration-test (after both tests complete)
6. docker-build (after integration-test)

### Example 4: TDD Loop with Human Signal

A workflow that demonstrates the retry loop.

```go
dagInput := DAGWorkflowInput{
  WorkflowID: "tdd-loop-example",
  Branch:     "feature/new-auth",
  Tasks: []Task{
    {
      Name:    "compile",
      Command: "go build ./...",
      Deps:    [],
    },
    {
      Name:    "unit-tests",
      Command: "go test -unit ./...",
      Deps:    []string{"compile"},
    },
    {
      Name:    "lint",
      Command: "golangci-lint run",
      Deps:    []string{"compile"},
    },
  },
}
```

Simulation:

```
ATTEMPT 1:
  compile ✅ (successful)
  unit-tests ✅ (successful)
  lint ❌ (4 style violations found)
  → DAG fails, waiting for FixApplied signal

[Engineer reads lint errors, fixes style violations, commits]

Signal: client.SignalWorkflow(..., "FixApplied", "Fixed style violations")

ATTEMPT 2:
  compile ✅ (successful)
  unit-tests ✅ (successful)
  lint ✅ (now passing)
  → DAG succeeds! Workflow returns nil
```

Temporal logs would show:

```
TDD Cycle Start attempt=1
Starting task name=compile
Task completed name=compile output="Built successfully"
Starting task name=unit-tests
Task completed name=unit-tests output="20 tests passed"
Starting task name=lint
Task failed name=lint error="style violations: 4"
TDD Cycle Failed attempt=1 error="tasks failed: [lint]"
Waiting for 'FixApplied' signal to retry...
Received FixApplied signal message="Fixed style violations"
TDD Cycle Start attempt=2
Starting task name=compile
Task completed name=compile output="Built successfully"
Starting task name=unit-tests
Task completed name=unit-tests output="20 tests passed"
Starting task name=lint
Task completed name=lint output="All checks passed"
TDD Cycle Succeeded! attempts=2
DAG Execution Complete tasksCompleted=3
```

## Advanced Features

### Activity Options

The DAG workflow configures Temporal activities with:

```go
ao := workflow.ActivityOptions{
  StartToCloseTimeout: 10 * time.Minute,   // Max time per task
  HeartbeatTimeout:    30 * time.Second,   // Max time without heartbeat
  RetryPolicy: &temporal.RetryPolicy{
    InitialInterval:    1 * time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    30 * time.Second,
    MaximumAttempts:    3,                // Retry 3 times on failure
  },
}
```

Each task gets:
- 10 minute timeout (adjust for your tasks)
- Automatic retry with exponential backoff
- Heartbeat monitoring

### Deadlock Detection

If no tasks are runnable but the DAG isn't complete, the workflow fails immediately:

```go
if len(pendingFutures) > 0 {
  selector.Select(ctx)
} else if len(completed) < len(tasks) {
  return fmt.Errorf("DAG stalled - no tasks runnable")  // Deadlock!
}
```

This prevents infinite hangs from circular dependency bugs.

### Selective Task Failure

When a task fails:
1. That task is marked as failed
2. Other pending tasks continue to completion
3. Once all pending tasks finish, DAG aborts

This allows you to see all failures at once rather than stopping immediately.

## Troubleshooting

### "Cycle detected in DAG"

**Cause**: Your task dependencies form a circle

**Solution**:
1. Draw your DAG on paper
2. Check for circular dependencies
3. Remove the problematic dependency edge
4. Re-run the workflow

### "DAG stalled - no tasks runnable"

**Cause**: All completed tasks are finished, but some tasks can't run (impossible dependencies)

**Solution**:
1. Check if all dependencies are present in task names
2. Verify no typos in dependency references
3. Ensure the DAG is acyclic

### Workflow hangs after task failure

**Cause**: Workflow waiting for "FixApplied" signal

**Solution**:
1. Fix the failed task in your code
2. Send the signal:
   ```go
   client.SignalWorkflow(ctx, workflowID, "", "FixApplied", "Fix applied")
   ```
3. DAG will retry from the beginning

### "Task failed: could not execute command"

**Cause**: The shell command returned an error

**Solution**:
1. Check the command syntax (test locally first)
2. Verify the command exists in the execution environment
3. Check working directory and environment variables
4. Look at the task output for details

## Best Practices

1. **Keep commands simple** - Use Makefiles or shell scripts for complex logic
2. **Fail fast** - Add `set -e` to shell scripts to stop on first error
3. **Log output** - Include useful logging in your commands
4. **Order your dependencies correctly** - Verify execution order with toposort
5. **Test locally first** - Run commands manually before adding to DAG
6. **Use meaningful task names** - Names are shown in logs and signals
7. **Set realistic timeouts** - Adjust StartToCloseTimeout for your tasks
8. **Monitor attempt count** - Track how many retries before success
9. **Document your DAG** - Comment why tasks depend on each other
10. **Handle partial failures** - Don't assume all tasks complete in TDD loop

## References

- [Temporal Workflow Execution](https://docs.temporal.io/workflows)
- [Go Topological Sort](https://github.com/gammazero/toposort)
- [bitfield/script - Shell Operations](https://github.com/bitfield/script)
- [Directed Acyclic Graph (Wikipedia)](https://en.wikipedia.org/wiki/Directed_acyclic_graph)

## See Also

- [TCR-WORKFLOW.md](./TCR-WORKFLOW.md) - Test-Commit-Revert pattern for single tasks
- [AGENTS.md](../AGENTS.md) - Multi-agent coordination with DAGs
- [README.md](../README.md) - Project overview
