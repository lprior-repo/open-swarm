# TCR Benchmark Quick Start Guide

Compare **Basic TCR** vs **Enhanced TCR** workflows to validate the hypothesis:
> **Enhanced Process > Raw Agent Intelligence**

## Prerequisites

1. **Temporal Server** running:
   ```bash
   make docker-up
   ```

2. **Temporal Worker** running in a separate terminal:
   ```bash
   make run-worker
   ```

3. **OpenCode** installed and in PATH:
   ```bash
   which opencode
   # Should show: /path/to/opencode
   ```

## Quick Start: Run Your First Benchmark

### Step 1: Test Basic Strategy

```bash
make run-benchmark \
  STRATEGY=basic \
  RUNS=5 \
  PROMPT="Implement a thread-safe LRU cache in pkg/cache/lru.go with Get, Put, and Evict methods"
```

**Expected Output:**
```
🚀 Starting BASIC Benchmark (5 runs)...
⏳ Workflow ID: bench-basic-1704931200

==========================================
📊 RESULTS: BASIC
==========================================
Runs:         5
✅ Success:   3 (60.0%)
❌ Failed:    2 (40.0%)
⏱️  Total Time: 12m30s
⏱️  Avg Time:  2m30s
------------------------------------------
```

### Step 2: Test Enhanced Strategy

```bash
make run-benchmark \
  STRATEGY=enhanced \
  RUNS=5 \
  PROMPT="Implement a thread-safe LRU cache in pkg/cache/lru.go with Get, Put, and Evict methods"
```

**Expected Output:**
```
🚀 Starting ENHANCED Benchmark (5 runs)...
⏳ Workflow ID: bench-enhanced-1704931300

==========================================
📊 RESULTS: ENHANCED
==========================================
Runs:         5
✅ Success:   5 (100.0%)
❌ Failed:    0 (0.0%)
⏱️  Total Time: 45m15s
⏱️  Avg Time:  9m3s
------------------------------------------
```

### Step 3: Compare Results

| Metric | Basic | Enhanced | Winner |
|--------|-------|----------|--------|
| Success Rate | 60% | 100% | 🏆 Enhanced |
| Avg Time | 2m30s | 9m3s | 🏆 Basic |
| False Positives | High | Low | 🏆 Enhanced |
| **Best For** | Simple tasks | Complex TDD tasks | - |

## Common Benchmark Scenarios

### Scenario 1: Simple Implementation (Basic Wins)

```bash
# Task: Add a simple method
make run-benchmark \
  STRATEGY=basic \
  RUNS=10 \
  PROMPT="Add a String() method to internal/model/user.go that returns name and email"
```

**Prediction:** High success rate, fast execution.

### Scenario 2: TDD-Required Task (Enhanced Wins)

```bash
# Task: Complex algorithm with edge cases
make run-benchmark \
  STRATEGY=enhanced \
  RUNS=5 \
  PROMPT="Implement a Leaky Bucket rate limiter with configurable rate and burst capacity. Must handle concurrent requests and edge cases."
```

**Prediction:** Enhanced catches edge cases via TDD gates.

### Scenario 3: Error-Prone Task (Enhanced Wins)

```bash
# Task: Concurrency-heavy code
make run-benchmark \
  STRATEGY=enhanced \
  RUNS=8 \
  PROMPT="Implement a concurrent-safe singleton pattern with lazy initialization, panic recovery, and graceful shutdown"
```

**Prediction:** Enhanced detects race conditions via multi-review gate.

## Monitoring Progress

### Temporal Web UI

1. Open http://localhost:8081
2. Search for workflow ID: `bench-basic-*` or `bench-enhanced-*`
3. View live execution:
   - Individual run progress
   - Child workflow details
   - Error stack traces

### Terminal Output

Watch for:
- ✅ Run completions
- ❌ Failures with error messages
- ⏱️ Duration tracking

## Interpreting Results

### Success Rate Analysis

| Rate | Interpretation | Action |
|------|----------------|--------|
| >80% | Strategy excels at this task | ✅ Use in production |
| 50-80% | Inconsistent, needs tuning | ⚠️ Review failures |
| <50% | Strategy unsuitable | ❌ Use alternative |

### Time Analysis

- **Basic**: Typically 1-5 minutes (no gates)
- **Enhanced**: Typically 5-15 minutes (6 gates)

### Cost-Benefit

```
Enhanced = 3-5x slower BUT 2-3x higher success rate
```

For production: **Reliability > Speed**

## Troubleshooting

### "Temporal connection failed"
```bash
# Start Temporal
make docker-up

# Verify
curl http://localhost:7233
```

### "No workers available"
```bash
# Start worker in separate terminal
make run-worker
```

### "OpenCode not found"
```bash
# Install OpenCode
curl -fsSL https://opencode.ai/install | bash

# Verify
opencode --version
```

### Low success rates on both strategies
```bash
# Check git repository
git status  # Should be clean

# Check worktree directory
ls -la ./worktrees/  # Should exist and be writable

# Check ports available
netstat -an | grep 800[0-9]  # Should show available ports
```

## Advanced Usage

### Custom Branch

```bash
make run-benchmark \
  STRATEGY=enhanced \
  RUNS=3 \
  PROMPT="Your task" \
  BRANCH=feature/my-branch
```

### High Concurrency

```bash
# Run 20 benchmarks
./bin/benchmark-tcr \
  -strategy enhanced \
  -runs 20 \
  -prompt "Your task" \
  -concurrency 10  # Limit to 10 concurrent runs
```

### Side-by-Side Comparison

**Terminal 1:**
```bash
make run-benchmark STRATEGY=basic RUNS=10 PROMPT="Your task"
```

**Terminal 2:**
```bash
make run-benchmark STRATEGY=enhanced RUNS=10 PROMPT="Your task"
```

## Next Steps

1. **Run 5+ benchmarks** with different task types
2. **Collect data** in a spreadsheet:
   - Task type
   - Basic success %
   - Enhanced success %
   - Time difference
3. **Identify patterns**: When does Enhanced justify the time cost?
4. **Document findings** for team decision-making

## Related Documentation

- [Full Benchmark README](../cmd/benchmark-tcr/README.md)
- [Enhanced TCR Workflow](./workflows/enhanced-tcr.md)
- [Basic TCR Workflow](./workflows/basic-tcr.md)
- [Temporal Architecture](./architecture/temporal.md)

## Example Benchmark Script

Save as `benchmark-suite.sh`:

```bash
#!/bin/bash
set -euo pipefail

PROMPTS=(
  "Add logging to internal/api/handler.go"
  "Implement binary search in pkg/search/binary.go"
  "Add rate limiting middleware with 100 req/min limit"
  "Implement thread-safe cache with TTL support"
)

for prompt in "${PROMPTS[@]}"; do
  echo "Testing: $prompt"
  
  make run-benchmark STRATEGY=basic RUNS=5 PROMPT="$prompt"
  make run-benchmark STRATEGY=enhanced RUNS=5 PROMPT="$prompt"
  
  echo "---"
done
```

Run with:
```bash
chmod +x benchmark-suite.sh
./benchmark-suite.sh
```

---

**Ready to prove the hypothesis? Start benchmarking!** 🚀