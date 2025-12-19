# Enhanced TCR Workflow - Execution Guide

This guide shows how to run the Enhanced TCR workflow and see actual output.

## Architecture Overview

The Enhanced TCR workflow implements a 6-gate Test-Commit-Revert pattern:

```
Bootstrap Cell
    ↓
Acquire File Locks
    ↓
Gate 1: Generate Tests → Gate 2: Lint Tests → Gate 3: Verify Tests Fail (RED)
    ↓
Gate 4: Generate Implementation
    ↓
┌─→ Gate 5: Verify Tests Pass (GREEN)
│   ├─ Failure? → Targeted Fix (within same state)
│   └─ Max fixes exceeded? → Full Regeneration (go back to Gate 4)
│
├─→ Gate 6: Multi-Reviewer Approval
│   ├─ Failure? → Targeted Fix (within same state)
│   └─ Max fixes exceeded? → Full Regeneration (go back to Gate 4)
│
└─→ All Gates Passed?
    ├─ YES → Commit Changes
    └─ NO  → Revert & Retry or Fail
```

## Prerequisites

### 1. Start Temporal Server

```bash
# Using Docker (recommended)
docker run --name temporal \
  -p 7233:7233 \
  -p 8080:8080 \
  -p 6379:6379 \
  -p 9042:9042 \
  -p 3306:3306 \
  temporalio/auto-setup:latest

# Or using Temporal CLI
temporal server start-dev
```

### 2. Build the Project

```bash
cd /home/lewis/src/open-swarm

# Build temporal worker
go build -o temporal-worker ./cmd/temporal-worker

# Build workflow runner
go build -o run-tcr ./cmd/run-tcr
```

## Running the Workflow

### Terminal 1: Start the Temporal Worker

```bash
./temporal-worker
```

Expected output:
```
🚀 Reactor-SDK Temporal Worker v6.1.0
🔭 Initializing OpenTelemetry tracing...
🔧 Initializing global managers...
✅ Connected to Temporal server
📋 Registered workflows and activities
⚙️  Worker listening on task queue: reactor-task-queue
```

### Terminal 2: Execute the Workflow

```bash
# Basic execution
./run-tcr -task my-feature-001

# With custom parameters
./run-tcr \
  -task my-feature-001 \
  -cell cell-prod-001 \
  -branch feature/new-api \
  -desc "Add REST endpoint for user profiles" \
  -criteria "API returns 200, all tests pass, reviewers approve" \
  -retries 2 \
  -fixes 5 \
  -reviewers 3
```

## Example Output

### Successful Run (All Gates Pass)

```
================================================================================
🚀 Enhanced TCR Workflow - Real Execution
================================================================================
Task ID:           my-feature-001
Cell ID:           cell-001
Branch:            main
Description:       Add REST endpoint for user profiles
Max Retries:       2
Max Fix Attempts:  5
Reviewers:         2
================================================================================

📋 Starting workflow execution...
✅ Workflow started with ID: my-feature-001

⏳ Waiting for workflow completion...

================================================================================
📊 Workflow Results
================================================================================
Status:          true
Error:
Files Changed:   3 files

Gate Execution Details:
--------------------------------------------------------------------------------
1. GenTest ✅ PASS (1.234s)
2. LintTest ✅ PASS (0.456s)
3. VerifyRED ✅ PASS (0.789s)
4. GenImpl ✅ PASS (2.345s)
5. VerifyGREEN ✅ PASS (1.234s)
6. MultiReview ✅ PASS (3.456s)
================================================================================
🎉 Workflow completed successfully!
================================================================================
```

### Failed Run (Gate Failure with Retry)

```
================================================================================
📊 Workflow Results
================================================================================
Status:          false
Error:           VerifyGREEN failed after 1 regen + 2 fix attempts: tests failed
Files Changed:   2 files

Gate Execution Details:
--------------------------------------------------------------------------------
1. GenTest ✅ PASS (1.234s)
2. LintTest ✅ PASS (0.456s)
3. VerifyRED ✅ PASS (0.789s)
4. GenImpl ✅ PASS (2.345s)
5. VerifyGREEN ❌ FAIL (1.234s)
   Error: 3 tests failed
6. FixFromFeedback ✅ PASS (0.567s)
5. VerifyGREEN ❌ FAIL (1.123s)
   Error: 2 tests still failing
6. FixFromFeedback ✅ PASS (0.612s)
4. GenImpl ✅ PASS (2.456s)  [REGENERATION ATTEMPT 1]
5. VerifyGREEN ❌ FAIL (1.234s)
   Error: tests failed
================================================================================
⚠️  Workflow failed - check errors above
================================================================================
```

## Monitoring Workflow Execution

### Via Temporal UI

Open http://localhost:8080 in your browser to see:
- Workflow execution timeline
- Activity execution logs
- Input/output data
- Retry history
- State transitions

### Via CLI

```bash
# List all workflows
temporal workflow list

# Describe specific workflow
temporal workflow describe \
  --workflow-id my-feature-001

# Show workflow history
temporal workflow show-history \
  --workflow-id my-feature-001

# Query workflow state
temporal workflow query \
  --workflow-id my-feature-001 \
  --query-type StateQuery
```

## Understanding the Gate Flow

### Test Generation Phase (Gates 1-3)
- **Gate 1 - GenTest**: AI generates test cases from acceptance criteria
- **Gate 2 - LintTest**: Verifies test syntax and style
- **Gate 3 - VerifyRED**: Confirms tests fail (RED state)
- **No retries**: These are foundational - failure terminates workflow

### Implementation Phase (Gates 4-6) - WITH RETRIES
- **Gate 4 - GenImpl**: AI generates implementation code
- **Gate 5 - VerifyGREEN**: Runs tests against implementation (GREEN state)
  - Fails? → Apply targeted fixes
  - Too many fixes? → Full regeneration (back to Gate 4)

- **Gate 6 - MultiReview**: Multiple reviewers evaluate code
  - Rejects? → Apply feedback fixes
  - Too many rejections? → Full regeneration (back to Gate 4)

### Retry Strategy

The workflow uses a **two-tier retry system**:

1. **Targeted Fixes** (within same gate):
   - Extract error feedback from gate failure
   - Apply minimal, focused changes
   - Preserves working code
   - Max attempts: configurable (default 5)

2. **Full Regeneration** (restart from Gate 4):
   - Regenerate entire implementation from scratch
   - Uses feedback from previous attempts
   - Max attempts: configurable (default 2)

## Test Coverage

The workflow is validated with 29 comprehensive tests:

```bash
# Run all tests
go test ./internal/temporal -v

# Run only E2E integration tests
go test ./internal/temporal -run "TestEnhancedTCR_" -v

# Results: 575+ tests passing
```

## Environment Variables

```bash
# Temporal Server Configuration
TEMPORAL_HOST_PORT=localhost:7233

# OpenTelemetry Tracing
OTEL_COLLECTOR_URL=localhost:4317
OTEL_SERVICE_NAME=open-swarm-tcr

# Cell Configuration
CELL_PORT_MIN=8000
CELL_PORT_MAX=9000
WORKTREE_DIR=./worktrees
```

## Troubleshooting

### Temporal Server Not Running
```bash
Error: Unable to connect to Temporal server: connection refused

Solution: Start Temporal server first (see Prerequisites)
```

### Worker Not Processing Tasks
```bash
Check temporal worker is listening on reactor-task-queue:
./temporal-worker
```

### Workflow Timeouts
Increase timeout in run-tcr:
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
```

## Next Steps

1. **Modify activities** in `internal/temporal/activities_enhanced.go` to integrate real AI agents
2. **Configure retry policies** by adjusting MaxRetries and MaxFixAttempts
3. **Integrate with CI/CD** to automatically run on pull requests
4. **Monitor performance** using Temporal UI dashboards
5. **Extend workflow** with additional gates or validation steps
