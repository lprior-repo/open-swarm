# TCR Workflow: Test-Commit-Revert Pattern

**Version:** 1.0.0
**Updated:** 2025-12-12
**Status:** Production Ready

## Overview

The Test-Commit-Revert (TCR) workflow is an automated execution pattern designed to safely execute development tasks within isolated cells and only persist changes if tests pass. It follows a strict sequence:

1. **Bootstrap** - Create isolated execution environment
2. **Execute** - Run the development task
3. **Test** - Verify work with test suite
4. **Commit or Revert** - Persist successful changes or discard failures
5. **Teardown** - Clean up resources

This pattern is ideal for:
- Autonomous agent-driven development
- Feature implementation with test-driven verification
- Bug fixes that must maintain test coverage
- Parallel task execution without conflicts
- Safe code generation and modification

## Architecture

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    TCR Workflow Starts                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  BOOTSTRAP CELL              │
        ├──────────────────────────────┤
        │ • Allocate port (8000-9000)  │
        │ • Create Git worktree        │
        │ • Start OpenCode server      │
        │ • Create SDK client          │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  EXECUTE TASK                │
        ├──────────────────────────────┤
        │ • Send prompt to agent       │
        │ • Agent modifies files       │
        │ • Track changed files        │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  RUN TESTS                   │
        ├──────────────────────────────┤
        │ • Execute test command       │
        │ • Parse test results         │
        │ • Return PASS/FAIL           │
        └──────────────┬───────────────┘
                       │
          ┌────────────┴────────────┐
          │                         │
    ┌─────▼──────┐         ┌────────▼──────┐
    │  COMMIT    │         │  REVERT       │
    ├────────────┤         ├───────────────┤
    │ Tests PASS │         │ Tests FAILED  │
    │            │         │               │
    │ • Git      │         │ • git reset   │
    │   commit   │         │   --hard      │
    │ • Record   │         │ • Discard all │
    │   success  │         │   changes     │
    └─────┬──────┘         └────────┬──────┘
          │                         │
          └────────────┬────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  TEARDOWN CELL               │
        ├──────────────────────────────┤
        │ • Kill OpenCode server       │
        │ • Remove Git worktree        │
        │ • Release port               │
        │ • Close SDK client           │
        └──────────────┬───────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  Workflow Complete           │
        │  Result: SUCCESS or FAILURE  │
        └──────────────────────────────┘
```

### Component Interaction Diagram

```
┌────────────────────────────────────────────────────────────┐
│                    Temporal Workflow                        │
│  (TCRWorkflow - Orchestrator & State Machine)              │
└─────┬──────────────────────────────────────────────────────┘
      │
      │ Coordinates Activities:
      │
      ├─→ BootstrapCell Activity
      │   ├─→ Port Manager: Allocate 8000-9000
      │   ├─→ Git Worktree Manager: Create worktree
      │   ├─→ Server Manager: Start 'opencode serve'
      │   └─→ Client Factory: Create SDK client
      │
      ├─→ ExecuteTask Activity
      │   ├─→ SDK Client: Send prompt
      │   ├─→ Agent: Modify files in worktree
      │   └─→ Result Handler: Track changes
      │
      ├─→ RunTests Activity
      │   ├─→ SDK Client: Execute test command
      │   ├─→ Shell: Run 'go test ./...'
      │   └─→ Result Handler: Parse output
      │
      ├─→ CommitChanges Activity
      │   └─→ Git: Commit to worktree branch
      │
      ├─→ RevertChanges Activity
      │   └─→ Git: Reset --hard to HEAD
      │
      └─→ TeardownCell Activity
          ├─→ Process Manager: Kill server
          ├─→ Git Worktree Manager: Remove worktree
          └─→ Port Manager: Release port
```

### Core Data Structures

```go
// TCRWorkflowInput - Workflow parameters
type TCRWorkflowInput struct {
    CellID      string // Unique cell identifier
    Branch      string // Git branch to work on
    TaskID      string // Unique task identifier
    Description string // Task description
    Prompt      string // The actual task prompt for agent
}

// TCRWorkflowResult - Workflow outcome
type TCRWorkflowResult struct {
    Success      bool     // Overall success (tests passed)
    TestsPassed  bool     // Test execution result
    FilesChanged []string // List of modified files
    Error        string   // Error message if failed
}

// BootstrapOutput - Cell configuration (serializable)
type BootstrapOutput struct {
    CellID       string // Cell identifier
    Port         int    // OpenCode server port
    WorktreeID   string // Git worktree identifier
    WorktreePath string // Path to worktree
    BaseURL      string // Server base URL
    ServerPID    int    // Server process ID
}

// TaskOutput - Execution result
type TaskOutput struct {
    Success       bool     // Execution succeeded
    Output        string   // Command output
    FilesModified []string // Changed files
    ErrorMessage  string   // Error description
}
```

## Usage

### Basic Invocation

Using the Reactor Client CLI:

```bash
# Submit a TCR workflow
reactor-client \
  --workflow tcr \
  --task TASK-001 \
  --desc "Add user authentication" \
  --prompt "Implement JWT-based auth in pkg/auth/jwt.go" \
  --branch main
```

### Output

```
✅ Workflow started
   ID: reactor-TASK-001
   RunID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
   Web UI: http://localhost:8233/namespaces/default/workflows/reactor-TASK-001

⏳ Waiting for workflow to complete...

✅ Workflow succeeded!
   Tests: PASSED
   Files changed: [pkg/auth/jwt.go, pkg/auth/jwt_test.go]
```

### Monitoring Progress

**Temporal Web UI:**
- Navigate to `http://localhost:8233/namespaces/default/workflows/reactor-TASK-001`
- Watch execution timeline
- View activity logs and heartbeats

**Logs:**
```bash
# View workflow logs
temporal workflow show --workflow-id reactor-TASK-001

# Stream logs in real-time
temporal workflow show --workflow-id reactor-TASK-001 --follow
```

## Parameters

### TCRWorkflowInput Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `CellID` | string | Yes | Unique identifier for the isolated execution cell. Example: `"primary"`, `"worker-1"` |
| `Branch` | string | Yes | Git branch to work on. Example: `"main"`, `"feature/auth"` |
| `TaskID` | string | Yes | Unique task identifier for tracking. Example: `"TASK-001"`, `"bd-xyz123"` |
| `Description` | string | Yes | Human-readable task description. Used in commit messages. |
| `Prompt` | string | Yes | The actual task prompt sent to the AI agent. Include specific instructions, file paths, and requirements. |

### Configuration

**Timeouts (hardcoded in workflow):**
```go
StartToCloseTimeout: 10 * time.Minute  // Total activity timeout
HeartbeatTimeout:   30 * time.Second   // Heartbeat interval
```

**Retry Policy:**
```go
MaximumAttempts: 1  // Don't retry non-idempotent operations
```

**Available Ports:**
- Range: 8000-9000 (1000 available ports)
- Allocation: First-fit strategy
- Release: Automatic on cell teardown

## Examples

### Example 1: Simple Feature Implementation

**Scenario:** Implement a new feature with TDD

```bash
reactor-client \
  --workflow tcr \
  --task AUTH-001 \
  --desc "Add JWT token validation" \
  --prompt "Implement JWT token validation middleware in pkg/middleware/jwt.go.
    Create tests in pkg/middleware/jwt_test.go.
    Ensure all tests pass before committing." \
  --branch feature/auth
```

**Expected Flow:**
1. Cell creates isolated worktree on `feature/auth`
2. Agent implements JWT validation
3. Agent writes tests
4. Tests run → PASS
5. Changes committed to worktree
6. Cell torn down
7. Developer merges `feature/auth` to `main`

**Result:**
```
✅ Workflow succeeded!
   Tests: PASSED
   Files changed: [pkg/middleware/jwt.go, pkg/middleware/jwt_test.go]
```

### Example 2: Bug Fix with Regression Testing

**Scenario:** Fix a critical bug while ensuring no regressions

```bash
reactor-client \
  --workflow tcr \
  --task BUG-042 \
  --desc "Fix SQL injection in user search" \
  --prompt "Fix the SQL injection vulnerability in internal/db/user_search.go.
    Use parameterized queries.
    Ensure all existing tests still pass.
    Add a test case for the SQL injection scenario." \
  --branch bugfix/sql-injection
```

**Expected Flow:**
1. Cell creates worktree from `bugfix/sql-injection` (which already branches from `main`)
2. Agent analyzes vulnerability and implements fix
3. Agent adds test for SQL injection scenario
4. All tests run → PASS (existing + new)
5. Changes auto-committed
6. Cell cleaned up

**Result:**
```
✅ Workflow succeeded!
   Tests: PASSED
   Files changed: [internal/db/user_search.go, internal/db/user_search_test.go]
```

### Example 3: Failed Attempt (Tests Fail)

**Scenario:** First implementation attempt fails tests

```bash
reactor-client \
  --workflow tcr \
  --task API-007 \
  --desc "Add pagination to user list endpoint" \
  --prompt "Implement pagination in cmd/api/handlers/users.go.
    Parameters: page (int, default 1), pageSize (int, default 20, max 100).
    Tests in cmd/api/handlers/users_test.go must pass." \
  --branch feature/pagination
```

**Expected Flow:**
1. Cell creates worktree
2. Agent implements pagination but makes an error (off-by-one in limit calculation)
3. Agent runs tests
4. Tests run → FAIL (test_pagination_page_two fails)
5. Changes automatically reverted (git reset --hard)
6. Cell torn down
7. Developer can investigate or retry

**Result:**
```
❌ Workflow failed
   Error: tests failed to run: exit code 1
```

**Developer Action:**
- Review agent output/logs
- Adjust prompt with more specific instructions
- Retry with corrected requirements

### Example 4: Parallel Execution (Multiple Tasks)

**Scenario:** Run three independent features in parallel

```bash
# Terminal 1
reactor-client --workflow tcr --task AUTH-001 --prompt "..." --branch feature/auth

# Terminal 2
reactor-client --workflow tcr --task CACHE-001 --prompt "..." --branch feature/cache

# Terminal 3
reactor-client --workflow tcr --task LOGGING-001 --prompt "..." --branch feature/logging
```

**Execution:**
```
Cell-1 (Port 8000) → TASK AUTH-001
Cell-2 (Port 8001) → TASK CACHE-001
Cell-3 (Port 8002) → TASK LOGGING-001

All three execute simultaneously in isolated environments.
No conflicts between modifications.
```

**Results:**
```
✅ AUTH-001 completed (commit to feature/auth)
✅ CACHE-001 completed (commit to feature/cache)
✅ LOGGING-001 completed (commit to feature/logging)
```

## Troubleshooting

### Workflow Fails with "bootstrap failed"

**Symptoms:**
```
❌ Workflow failed
   Error: bootstrap failed: unable to allocate port
```

**Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| All ports (8000-9000) in use | `pkill opencode` to kill zombie servers, then retry |
| OpenCode not installed | Run `curl -fsSL https://opencode.ai/install \| bash` |
| Permission denied | Ensure write access to worktree directory |
| Git not configured | Run `git config --global user.email "bot@example.com"` |

**Debug Steps:**
```bash
# Check port availability
netstat -tlnp | grep -E "800[0-9]"

# List active OpenCode processes
ps aux | grep "opencode serve"

# Verify Git setup
git config user.name
git config user.email

# Test OpenCode manually
opencode serve --port 8000 --dir ./test-worktree
```

### Workflow Succeeds but Tests Fail

**Symptoms:**
```
❌ Workflow failed
   Error: tests failed
```

**Likely Causes:**
1. Test suite has flaky tests
2. Agent didn't understand requirements fully
3. Environment setup incomplete
4. Test timeout too short

**Solutions:**

1. **Improve Prompt Specificity:**
   ```bash
   # Before (vague)
   --prompt "Add user validation"

   # After (specific)
   --prompt "Add email validation to User struct in pkg/user/types.go.
     Must validate email format using regex pattern /.+@.+\.+/.
     Must trim whitespace before validation.
     Add tests in pkg/user/types_test.go covering:
     - valid emails
     - invalid emails
     - empty string
     - whitespace handling"
   ```

2. **Check Test Output:**
   ```bash
   # View test failure details in Temporal UI
   http://localhost:8233/namespaces/default/workflows/reactor-TASK-001
   # Scroll to "RunTests" activity and read output
   ```

3. **Verify Environment:**
   ```bash
   # Ensure test dependencies installed
   go mod download
   go test ./... -v
   ```

### "Server failed to become ready"

**Symptoms:**
```
❌ bootstrap failed: server failed to become ready
```

**Causes:**

| Cause | Solution |
|-------|----------|
| OpenCode slow to start | Increase healthcheck timeout (currently 10s) |
| System resources exhausted | Reduce concurrent workflows or add resources |
| OpenCode crash on startup | Check OpenCode logs: `tail -f ~/.opencode/logs` |

**Verify Manually:**
```bash
# Test OpenCode startup time
time opencode serve --port 8000 --dir /tmp/test

# Should complete within 2 seconds
# If slower, system is overloaded
```

### Worktree Already Exists

**Symptoms:**
```
❌ bootstrap failed: worktree already exists
```

**Causes:**
- Previous run didn't clean up
- Manual interruption left resources

**Solutions:**
```bash
# Clean up all worktrees
git worktree prune
git worktree remove ./worktrees/* --force

# Kill all OpenCode instances
pkill -f "opencode serve"

# Release any held ports
lsof -i :8000-9000 | awk 'NR>1 {print $2}' | xargs kill -9

# Retry workflow
reactor-client --workflow tcr --task TASK-001 ...
```

### Agent Output Seems Wrong or Incomplete

**Symptoms:**
- Files don't exist after execution
- Agent says "wrote file" but file isn't there
- Output truncated or missing

**Debug:**

1. **Check Activity Logs:**
   ```bash
   temporal activity describe --workflow-id reactor-TASK-001 --activity-id ExecuteTask
   ```

2. **View Full Output:**
   - Temporal Web UI → Workflow Details → ExecuteTask Activity
   - Check stderr and stdout capture

3. **Verify Agent Connection:**
   ```bash
   # Check if OpenCode server is running
   curl http://localhost:8000/health
   # Should return 200 OK
   ```

4. **Check Worktree State:**
   ```bash
   # During execution, inspect worktree directly
   ls -la ./worktrees/[worktree-id]
   git -C ./worktrees/[worktree-id] log --oneline
   ```

### Tests Pass but Commit Fails

**Symptoms:**
```
✅ Tests passed
❌ Commit failed: fatal: not a git repository
```

**Causes:**
- Git worktree corrupted
- Concurrent modification of same worktree

**Solutions:**
```bash
# Verify Git state
git -C ./worktrees/[id] status

# Check for concurrent access
lsof | grep worktrees

# Force cleanup and retry
rm -rf ./worktrees/[id]
reactor-client --workflow tcr --task TASK-001 ...
```

## Advanced Topics

### Custom Test Commands

The workflow currently runs `go test ./...` by default. To customize:

Edit `internal/temporal/activities_cell.go` - `RunTests` method:

```go
func (ca *CellActivities) RunTests(ctx context.Context, bootstrap *BootstrapOutput) (bool, error) {
    cell := ca.reconstructCell(bootstrap)

    // Run custom test command
    // Example: pytest for Python projects
    // return ca.activities.RunCommand(ctx, cell, "pytest ./tests")

    return ca.activities.RunTests(ctx, cell)
}
```

### Handling Non-Go Projects

Currently optimized for Go. To support other languages:

1. Create a wrapper script in repo root:
   ```bash
   #!/bin/bash
   # ./scripts/test.sh
   case $(git config -f .langconfig lang) in
     python) pytest ./tests ;;
     node) npm test ;;
     *) go test ./... ;;
   esac
   ```

2. Modify prompt to include:
   ```
   Execute tests using ./scripts/test.sh
   ```

### Monitoring Cell Resources

To add resource monitoring during execution:

```go
// In BootstrapCell activity
activity.RecordHeartbeat(ctx, HeartbeatDetails{
    Port: bootstrap.Port,
    PID: bootstrap.ServerPID,
    Memory: getProcessMemory(bootstrap.ServerPID),
    CPU: getProcessCPU(bootstrap.ServerPID),
})
```

### Retry Logic

Currently configured with `MaximumAttempts: 1` (no retries). To enable retries:

```go
RetryPolicy: &temporal.RetryPolicy{
    MaximumAttempts: 3,  // Retry up to 3 times
    BackoffCoefficient: 2.0,
    InitialInterval: 1 * time.Second,
}
```

**Warning:** Only safe for idempotent operations (bootstrap, teardown). Not safe for ExecuteTask.

## Best Practices

### 1. Write Clear, Specific Prompts

Bad:
```
--prompt "Add validation"
```

Good:
```
--prompt "Add email validation to User.Email field.
  Requirements:
  - Use regex pattern /^[^@]+@[^@]+\.[^@]+$/
  - Trim whitespace before validation
  - Return specific error for invalid format
  Tests required in user_test.go:
  - Valid email addresses
  - Invalid format
  - Whitespace handling
  - Edge cases (no @, multiple @)"
```

### 2. Include Test Expectations

Specify exactly what tests should verify:
```
--prompt "Implement pagination.
  Tests must verify:
  1. Default page=1, pageSize=20
  2. Custom page/pageSize values work
  3. pageSize max enforced at 100
  4. Total count returned correctly
  5. No duplicates across pages
  6. Requesting page beyond total returns empty"
```

### 3. Use Feature Branches

Each TCR workflow should work on isolated branches:
```bash
# Create feature branch first
git checkout -b feature/description

# Then run workflow on that branch
reactor-client --workflow tcr --branch feature/description ...
```

### 4. Monitor First Attempt

For new workflows, watch the first run:
```bash
# Terminal 1: Start workflow
reactor-client --workflow tcr --task NEW-001 --prompt "..."

# Terminal 2: Watch in UI
open "http://localhost:8233/namespaces/default/workflows/reactor-NEW-001"

# Terminal 3: Monitor logs
tail -f ~/.temporal/worker.log
```

### 5. Handle Partial Failures

If workflow fails partway through:

1. **Don't immediately retry** - Investigate why
2. **Review agent output** - What did it try?
3. **Check error logs** - Where exactly did it fail?
4. **Adjust prompt** - Provide more context/guidance
5. **Retry with improvements**

## Architecture Decisions

### Why Saga Pattern for Teardown?

The workflow uses a deferred cleanup with disconnected context:

```go
defer func() {
    disconnCtx, _ := workflow.NewDisconnectedContext(ctx)
    // Teardown even if workflow fails
    _ = workflow.ExecuteActivity(disconnCtx, ...)
}()
```

**Rationale:**
- Ensures cleanup happens even on failure
- Prevents zombie OpenCode processes
- Releases ports for reuse
- Cleans up Git worktrees

### Why No Retries on ExecuteTask?

```go
RetryPolicy: &temporal.RetryPolicy{
    MaximumAttempts: 1,  // No retries
}
```

**Rationale:**
- ExecuteTask modifies code (non-idempotent)
- Retrying could create duplicate files/changes
- Caller controls retry via new workflow submission
- Cleaner error handling and debugging

### Why 10-Minute Activity Timeout?

```go
StartToCloseTimeout: 10 * time.Minute
```

**Rationale:**
- Accounts for slow network/system
- Large language models (slow first token)
- Complex code generation (multiple files)
- Test suite execution time

Adjust if needed:
- Faster systems: 5 minutes
- Complex codebases: 20 minutes

## Performance Characteristics

### Typical Execution Times

| Phase | Duration | Notes |
|-------|----------|-------|
| Bootstrap | 2-3s | Port alloc, server startup, healthcheck |
| Execute | 30-120s | Depends on prompt complexity & LLM |
| Test | 5-30s | Depends on test suite size |
| Commit/Revert | 1s | Git operation |
| Teardown | 2-3s | Process cleanup, resource release |
| **Total** | **40-160s** | Typical range |

### Scalability

**Concurrent Workflows:**
- Limit: 50 concurrent cells (default)
- Port limit: 1000 (8000-9000)
- Memory: ~200MB per cell
- CPU: 1 core per cell under load

**Bottlenecks:**
1. LLM throughput (token generation rate)
2. Test execution time
3. System resources (ports, memory, file handles)

## Integration Examples

### With Beads Issue Tracking

```bash
# Get ready task from Beads
bd ready | head -1

# Extract task ID
TASK_ID=$(bd ready --json | jq -r '.[0].id')

# Submit TCR workflow
reactor-client \
  --workflow tcr \
  --task "$TASK_ID" \
  --desc "$(bd show $TASK_ID --json | jq -r .title)" \
  --prompt "$(bd show $TASK_ID --json | jq -r .description)"
```

### With Agent Mail Coordination

```bash
# After workflow completes
if [ $? -eq 0 ]; then
  am send -to coordinator \
    -subject "Task $TASK_ID: TCR workflow completed" \
    -body "Changes committed successfully"
else
  am send -to coordinator \
    -subject "Task $TASK_ID: TCR workflow failed" \
    -body "Tests failed - review in Temporal UI"
fi
```

## References

- [Temporal Documentation](https://docs.temporal.io/)
- [Go Temporal SDK](https://github.com/temporalio/sdk-go)
- [OpenCode Documentation](https://opencode.ai/docs/)
- [Git Worktrees](https://git-scm.com/docs/git-worktree)
- [TCR Kata](https://en.wikipedia.org/wiki/Test_commit_revert) (Original TCR pattern concept)

## Support

For issues with TCR workflow:

1. Check logs: `tail -f ~/.temporal/worker.log`
2. Review Temporal UI: `http://localhost:8233`
3. Verify OpenCode: `opencode --version`
4. Check Git: `git --version && git config --list`
5. File issue in Beads: `bd create "TCR issue description"`

---

**Document Version:** 1.0.0
**Last Updated:** 2025-12-12
**Maintained by:** Open Swarm Team
