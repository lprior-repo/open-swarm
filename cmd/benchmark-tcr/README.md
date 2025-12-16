# TCR Benchmark Tool

A CLI tool for benchmarking and comparing **Basic TCR** vs **Enhanced TCR** workflows in head-to-head performance evaluations.

## Purpose

This tool tests the hypothesis: **Enhanced Process > Raw Agent Intelligence**

- **Basic TCR**: Single agent, minimal validation (fast but error-prone)
- **Enhanced TCR**: Multi-gate validation, test-driven flow (slower but reliable)

## Installation

```bash
# Build the tool
go build -o bin/benchmark-tcr ./cmd/benchmark-tcr

# Or run directly
go run ./cmd/benchmark-tcr/main.go [flags]
```

## Prerequisites

1. **Temporal Server** running on `localhost:7233`
2. **Temporal Worker** running with benchmark workflows registered:
   ```bash
   go run ./cmd/temporal-worker/main.go
   ```
3. **OpenCode** installed and accessible in PATH
4. **Git** repository initialized

## Usage

### Basic Benchmark (Fast, Unvalidated)

```bash
go run ./cmd/benchmark-tcr/main.go \
  -strategy basic \
  -runs 5 \
  -prompt "Implement a thread-safe LRU cache in pkg/cache/lru.go"
```

### Enhanced Benchmark (Slow, Validated)

```bash
go run ./cmd/benchmark-tcr/main.go \
  -strategy enhanced \
  -runs 5 \
  -prompt "Implement a thread-safe LRU cache in pkg/cache/lru.go"
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-strategy` | string | `basic` | Workflow strategy: `basic` or `enhanced` |
| `-runs` | int | `3` | Number of parallel runs to execute |
| `-prompt` | string | *required* | Coding challenge prompt for the agent |
| `-branch` | string | `main` | Git branch to work on |
| `-concurrency` | int | `0` | Max concurrent runs (0 = unlimited) |

## Example Scenarios

### Scenario 1: Simple Implementation

Test a straightforward task where basic agent should excel:

```bash
# Basic Strategy
go run ./cmd/benchmark-tcr/main.go \
  -strategy basic \
  -runs 10 \
  -prompt "Add a String() method to pkg/model/user.go"

# Enhanced Strategy (for comparison)
go run ./cmd/benchmark-tcr/main.go \
  -strategy enhanced \
  -runs 10 \
  -prompt "Add a String() method to pkg/model/user.go"
```

**Expected Result**: Basic wins on speed, similar success rates.

### Scenario 2: Complex TDD Task

Test a task requiring test-first development:

```bash
# Basic Strategy (expected to struggle)
go run ./cmd/benchmark-tcr/main.go \
  -strategy basic \
  -runs 5 \
  -prompt "Implement a Leaky Bucket rate limiter with configurable rate and capacity. Must pass tests for burst handling and rate limiting accuracy."

# Enhanced Strategy (expected to excel)
go run ./cmd/benchmark-tcr/main.go \
  -strategy enhanced \
  -runs 5 \
  -prompt "Implement a Leaky Bucket rate limiter with configurable rate and capacity. Must pass tests for burst handling and rate limiting accuracy."
```

**Expected Result**: Enhanced has higher success rate due to TDD gates.

### Scenario 3: Error-Prone Task

Test a task with subtle edge cases:

```bash
# Basic Strategy
go run ./cmd/benchmark-tcr/main.go \
  -strategy basic \
  -runs 8 \
  -prompt "Implement concurrent-safe singleton pattern in pkg/singleton/manager.go with lazy initialization and panic recovery"

# Enhanced Strategy
go run ./cmd/benchmark-tcr/main.go \
  -strategy enhanced \
  -runs 8 \
  -prompt "Implement concurrent-safe singleton pattern in pkg/singleton/manager.go with lazy initialization and panic recovery"
```

**Expected Result**: Enhanced catches edge cases via multi-review gate.

## Output Format

```
🚀 Starting BASIC Benchmark (5 runs)...
⏳ Workflow ID: bench-basic-1704931200
⏳ Run ID: abc123...

==========================================
📊 RESULTS: BASIC
==========================================
Runs:         5
✅ Success:   3 (60.0%)
❌ Failed:    2 (40.0%)
⏱️  Total Time: 15m30s
⏱️  Avg Time:  3m6s
------------------------------------------

📋 Individual Run Results:
  ✅ Run #1: 2m45s - Files: 2
  ✅ Run #2: 3m10s - Files: 3
  ❌ Run #3: 4m20s - Error: tests failed: race condition detected
  ✅ Run #4: 2m55s - Files: 2
  ❌ Run #5: 3m20s - Error: tests failed: nil pointer dereference
==========================================
```

## Interpreting Results

### Success Rate
- **High (>80%)**: Strategy handles task well
- **Medium (50-80%)**: Strategy inconsistent, may need retries
- **Low (<50%)**: Strategy unsuitable for task complexity

### Average Time
- **Basic**: Typically 1-5 minutes per run
- **Enhanced**: Typically 5-15 minutes per run (6-gate validation)

### Files Changed
- More files = more comprehensive solution
- Zero files on failure = early gate rejection (good for Enhanced)

## Head-to-Head Comparison

Run both strategies in parallel terminals to compare:

**Terminal 1:**
```bash
go run ./cmd/benchmark-tcr/main.go -strategy basic -runs 10 -prompt "YOUR_PROMPT"
```

**Terminal 2:**
```bash
go run ./cmd/benchmark-tcr/main.go -strategy enhanced -runs 10 -prompt "YOUR_PROMPT"
```

### Comparison Metrics

| Metric | Basic (Expected) | Enhanced (Expected) |
|--------|------------------|---------------------|
| Success Rate | 40-70% | 80-100% |
| Avg Time | 2-5 min | 8-15 min |
| False Positives | High (commits broken code) | Low (gates catch issues) |
| Ideal Use Case | Simple, well-defined tasks | Complex, TDD-required tasks |

## Temporal Web UI

Monitor live progress at `http://localhost:8233`:
- View workflow execution details
- Inspect individual child workflows
- Debug failures with stack traces
- Replay failed workflows

## Troubleshooting

### "Temporal connection failed"
- Ensure Temporal server is running: `temporal server start-dev`
- Check `localhost:7233` is accessible

### "Workflow failed: no workers available"
- Start the Temporal worker: `go run ./cmd/temporal-worker/main.go`
- Verify worker is registered on task queue: `reactor-task-queue`

### "Prompt required"
- You must provide a `-prompt` flag with a coding challenge

### Low success rates on both strategies
- Check OpenCode is installed: `which opencode`
- Verify Git repository is clean: `git status`
- Ensure worktree directory is writable: `./worktrees/`

## Architecture

```
BenchmarkWorkflow
├── Input: Strategy, NumRuns, Prompt
├── For each run (parallel):
│   ├── Generate unique CellID
│   ├── Execute child workflow (TCR or Enhanced)
│   └── Collect result
└── Aggregate: Success/Failure counts, durations
```

Each run is an independent Temporal child workflow with isolated:
- Git worktree
- OpenCode server port
- Task execution context

## Performance Considerations

- **Runs**: More runs = better statistical significance (recommend 5-10)
- **Concurrency**: Limited by available ports (8000-9000 range)
- **Timeouts**: Each run has 30-minute timeout (adjust in code if needed)
- **Retries**: Disabled in benchmarks (no retries for fair comparison)

## Next Steps

After running benchmarks:

1. **Analyze Results**: Compare success rates and times
2. **Identify Patterns**: Which tasks favor which strategy?
3. **Tune Parameters**: Adjust gates, reviewers, timeouts
4. **Scale Testing**: Run larger benchmarks (50+ runs)
5. **Report Findings**: Document hypothesis validation

## Related Documentation

- [Enhanced TCR Workflow](../../docs/workflows/enhanced-tcr.md)
- [Basic TCR Workflow](../../docs/workflows/basic-tcr.md)
- [Temporal Architecture](../../docs/architecture/temporal.md)