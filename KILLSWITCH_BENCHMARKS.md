# Kill Switch Performance Benchmarks

This document describes the comprehensive benchmark suite for the kill switch functionality in the merge queue coordinator. The benchmarks measure performance characteristics including timing, resource usage, and scalability under various scenarios.

## Overview

The kill switch benchmarks test two main functions:
1. **Single Branch Kill**: `killFailedBranchWithTimeout()`
2. **Cascade Kill**: `killDependentBranchesWithTimeout()` and `killDependentBranchesRecursive()`

## Benchmark Categories

### 1. Single Branch Kill Performance

#### BenchmarkKillFailedBranchWithTimeout_SingleBranch
**Purpose**: Measure baseline performance of killing a single branch
- **What it measures**: End-to-end time to kill one branch
- **Expected range**: < 10ms (sub-millisecond on modern systems)
- **Significance**: Establishes baseline performance expectation
- **Uses**: Creates new branch per iteration, measures lock acquisition + status update + metrics increment

#### BenchmarkKillFailedBranchWithTimeout_IdempotentKill
**Purpose**: Measure overhead of idempotent kill operations
- **What it measures**: Time to kill an already-killed branch (no-op path)
- **Expected range**: < 1ms (much faster than fresh kill)
- **Significance**: Validates idempotency check is lightweight
- **Uses**: Pre-killed branch, tests fast-path execution

#### BenchmarkKillFailedBranchWithTimeout_KillWithTimeout
**Purpose**: Measure timeout handling overhead
- **What it measures**: Time to handle timeout scenario (graceful degradation)
- **Expected range**: ~1-2ms (timeout overhead)
- **Significance**: Quantifies cost of timeout context creation and selection
- **Uses**: 1ns timeout to consistently trigger timeout path

#### BenchmarkKillFailedBranchWithTimeout_MutexContention
**Purpose**: Measure performance under high contention
- **What it measures**: Kill operation time with 1000 existing branches
- **Expected range**: < 5ms (linear time with map size should be negligible)
- **Significance**: Validates map access remains O(1) even with many branches
- **Uses**: Pre-populates activeBranches with 1000 branches before benchmark

### 2. Multiple Branch Kill Performance

#### BenchmarkKillMultipleBranchesSequential
**Purpose**: Measure time to kill multiple branches one at a time
- **What it measures**: Sequential kill of 10 branches per iteration
- **Expected range**: < 100ms (10 * baseline + overhead)
- **Significance**: Validates linear scaling with number of kills
- **Uses**: Creates 10 branches, kills each sequentially

#### BenchmarkKillMultipleBranchesParallel
**Purpose**: Measure time to kill multiple branches concurrently
- **What it measures**: Parallel kill of 10 branches using goroutines
- **Expected range**: < 20ms (baseline + lock contention, faster than sequential)
- **Significance**: Validates concurrent performance under mutex contention
- **Uses**: Creates 10 branches, kills all in parallel with sync.WaitGroup

### 3. Cascade Kill Performance (Hierarchy Scenarios)

#### BenchmarkKillDependentBranchesWithTimeout_ShallowHierarchy
**Purpose**: Measure cascade kill performance with shallow tree
- **What it measures**: Kill parent + 3 children (2-level tree)
- **Expected range**: < 50ms
- **Tree structure**:
  ```
  parent
  ├── child-0
  ├── child-1
  └── child-2
  ```
- **Significance**: Baseline for cascade operation

#### BenchmarkKillDependentBranchesWithTimeout_DeepHierarchy
**Purpose**: Measure cascade kill performance with deep linear tree
- **What it measures**: Kill 10-level deep hierarchy (10 recursive calls)
- **Expected range**: 10-100ms (scales with depth)
- **Tree structure**: Linear chain: root → level-1 → level-2 → ... → level-10
- **Significance**: Validates timeout handling at depth, tests recursion performance

#### BenchmarkKillDependentBranchesWithTimeout_WideHierarchy
**Purpose**: Measure cascade kill performance with many children
- **What it measures**: Kill parent with 50 children (2-level tree)
- **Expected range**: < 100ms
- **Tree structure**:
  ```
  parent
  ├── child-0
  ├── child-1
  ├── ...
  └── child-49
  ```
- **Significance**: Tests performance with many branches at same level

#### BenchmarkKillDependentBranchesWithTimeout_ComplexHierarchy
**Purpose**: Measure cascade kill performance with balanced tree
- **What it measures**: Kill mixed hierarchy (8 nodes, 3 levels)
- **Expected range**: < 50ms
- **Tree structure**:
  ```
        root
       /    \
      l1-0  l1-1
      / \    / \
    l2-0 l2-1 l2-2 l2-3
     |
    l3-0
  ```
- **Significance**: Validates performance with mixed wide/deep structure

### 4. Protection Mechanism Overhead

#### BenchmarkKillProtectionMechanism_MutexAcquisition
**Purpose**: Measure pure mutex lock/unlock overhead
- **What it measures**: Time to acquire and release RWMutex lock
- **Expected range**: < 1µs per lock (nanoseconds on uncontended mutex)
- **Significance**: Establishes lock overhead baseline
- **Uses**: Minimal work inside lock to isolate lock time

#### BenchmarkKillProtectionMechanism_TimeoutContextCreation
**Purpose**: Measure context creation overhead
- **What it measures**: Time to create context.WithTimeout()
- **Expected range**: 1-10µs
- **Significance**: Validates timeout context is lightweight
- **Uses**: Creates and cancels context, measures creation time

#### BenchmarkKillProtectionMechanism_ChannelSelect
**Purpose**: Measure timeout select operation overhead
- **What it measures**: Time to check context timeout via select
- **Expected range**: < 1µs
- **Significance**: Validates timeout checking is efficient
- **Uses**: select on context.Done() channel

#### BenchmarkKillProtectionMechanism_IdempotencyCheck
**Purpose**: Measure idempotency check (branch status comparison)
- **What it measures**: Time to check if branch is already killed
- **Expected range**: < 1µs
- **Significance**: Validates idempotency overhead is negligible
- **Uses**: Lock + map lookup + status comparison on pre-killed branch

### 5. Memory Allocation Benchmarks

#### BenchmarkMemoryAllocation_BranchCreation
**Purpose**: Measure memory allocation for branch creation
- **What it measures**: Allocations per branch with Changes slice
- **Expected result**: ~1-2 allocations per branch (the struct + slice growth)
- **Significance**: Validates reasonable memory efficiency
- **Uses**: ReportAllocs() to show allocation counts

#### BenchmarkMemoryAllocation_KilledBranchMetadata
**Purpose**: Measure memory allocation for kill metadata
- **What it measures**: Allocations when adding kill timestamp and reason
- **Expected result**: ~2-3 allocations (time.Time + string)
- **Significance**: Validates kill operation metadata is lightweight

#### BenchmarkMemoryAllocation_HierarchyCreation
**Purpose**: Measure memory allocation for hierarchy structures
- **What it measures**: Allocations with ChildrenIDs slices
- **Expected result**: ~2-3 allocations per branch (struct + slice)
- **Significance**: Validates hierarchy memory cost is reasonable

### 6. Concurrent Operation Benchmarks

#### BenchmarkConcurrentKillOperations
**Purpose**: Measure concurrent kill performance with limited workers
- **What it measures**: Kill 100 branches with 10 concurrent workers
- **Expected range**: < 500ms
- **Significance**: Validates semaphore-controlled concurrency
- **Uses**: sync.WaitGroup + channel semaphore to limit goroutines

#### BenchmarkCascadeKillTiming
**Purpose**: Measure cascade kill performance on large binary tree
- **What it measures**: Kill complete binary tree (63 nodes, 7 levels)
- **Expected range**: 50-200ms
- **Tree structure**: Full binary tree
  ```
              root
             /    \
           n-1    n-2
          /  \    /  \
       n-3 n-4 n-5 n-6
       ... (7 levels total, 63 nodes)
  ```
- **Significance**: Realistic large-scale cascade scenario
- **Uses**: Recursive tree building, tests worst-case recursion + locking

## Running the Benchmarks

### Run all kill switch benchmarks
```bash
go test -bench=^BenchmarkKill -v ./internal/mergequeue/
```

### Run specific benchmark
```bash
go test -bench=^BenchmarkKillFailedBranchWithTimeout_SingleBranch$ -v ./internal/mergequeue/
```

### Run with memory allocation reporting
```bash
go test -bench=^BenchmarkMemory -v -benchmem ./internal/mergequeue/
```

### Run with custom duration
```bash
go test -bench=^BenchmarkKill -benchtime=10s -v ./internal/mergequeue/
```

### Run benchmarks on current code (baseline)
```bash
go test -bench=^BenchmarkKill -benchmem -count=5 -v ./internal/mergequeue/ | tee baseline.txt
```

### Compare with previous baseline
```bash
go test -bench=^BenchmarkKill -benchmem -count=5 -v ./internal/mergequeue/ > current.txt
benchstat baseline.txt current.txt
```

## Performance Expectations

### Latency Targets

| Operation | Baseline | With Timeout | With Contention |
|-----------|----------|--------------|-----------------|
| Single Kill | < 10µs | < 50µs | < 100µs |
| Idempotent Kill | < 5µs | < 10µs | < 20µs |
| Shallow Cascade (3 children) | < 100µs | < 200µs | < 500µs |
| Deep Cascade (10 levels) | < 1ms | < 2ms | < 5ms |
| Wide Cascade (50 children) | < 1ms | < 2ms | < 5ms |
| Complex Cascade (8 nodes) | < 500µs | < 1ms | < 2ms |

### Throughput Targets

| Operation | Sequential | Parallel (10 workers) |
|-----------|------------|----------------------|
| Kill 10 branches | ~1ms | ~100µs |
| Kill 100 branches | ~10ms | ~1ms |
| Kill 1000 branches | ~100ms | ~10ms |

### Memory Targets

| Operation | Allocations | Bytes per alloc |
|-----------|-------------|-----------------|
| Branch creation | ~2 | ~500-1000 |
| Kill metadata | ~1-2 | ~50-200 |
| Hierarchy (10 children) | ~2-3 | ~500-2000 |

## Optimization Guidelines

### When to Investigate Performance

1. **Single Kill > 100µs**: Indicates lock contention or platform issues
2. **Cascade Kill > 10ms**: For < 100 branches indicates algorithm inefficiency
3. **Concurrent Kills > 50% slower**: Indicates excessive lock contention
4. **Memory allocations increase**: Watch for unintended allocations

### Performance Tuning Knobs

1. **KillSwitchTimeout**: Longer timeout = less graceful degradation overhead
   - Default: 500ms
   - Tuning: Increase for slower systems, decrease for faster cleanup

2. **Cascade timeout multiplier**: Currently 10x KillSwitchTimeout
   - Tuning: Adjust based on hierarchy depth
   - Formula: `cascadeTimeout = killSwitchTimeout * (max_depth + 1)`

3. **Mutex strategy**: Currently using sync.RWMutex
   - Option: Switch to sync.Mutex if read-heavy pattern not beneficial
   - Measurement: Use benchmarks to detect lock contention

### Profiling Integration

For detailed performance analysis:

```bash
# CPU profiling
go test -bench=^BenchmarkCascadeKillTiming$ -cpuprofile=cpu.prof -v ./internal/mergequeue/
go tool pprof cpu.prof

# Memory profiling
go test -bench=^BenchmarkMemory -memprofile=mem.prof -v ./internal/mergequeue/
go tool pprof mem.prof

# Trace analysis (Go 1.11+)
go test -bench=^BenchmarkKill -trace=trace.out -v ./internal/mergequeue/
go tool trace trace.out
```

## Regression Detection

### Setting up continuous benchmarking

1. Save baseline on stable version:
   ```bash
   go test -bench=^BenchmarkKill -benchmem -count=5 ./internal/mergequeue/ > baseline.txt
   ```

2. After changes, compare:
   ```bash
   go test -bench=^BenchmarkKill -benchmem -count=5 ./internal/mergequeue/ > current.txt
   benchstat baseline.txt current.txt
   ```

3. **Alert if**: Any benchmark degrades > 10% in latency or > 20% in allocations

## Benchmark Maintenance

### When to Update Benchmarks

1. **New kill scenarios discovered**: Add corresponding benchmark
2. **Performance requirements change**: Update expected ranges
3. **Architecture changes**: Verify with new benchmarks
4. **Timeout behavior changes**: Update timeout-related benchmarks

### Benchmark Structure Guidelines

Each benchmark should:
1. Have clear Setup (b.StopTimer) and Benchmark (b.StartTimer) phases
2. Use unique identifiers to avoid map collisions
3. Clean up resources (defer statements)
4. Report relevant metrics (allocations, bytes)
5. Include comments explaining what is measured

## Integration with CI/CD

Recommended CI practices:

1. **Quick benchmarks** (< 30s total): Run on every PR
   ```bash
   go test -bench=^BenchmarkKillFailed -short ./internal/mergequeue/
   ```

2. **Full benchmarks** (2-5m total): Run on merge to main
   ```bash
   go test -bench=^BenchmarkKill -benchmem -count=5 ./internal/mergequeue/
   ```

3. **Store results**: Save benchmark results for historical tracking
4. **Trend analysis**: Track performance over time

## See Also

- [KILLSWITCH.md](KILLSWITCH.md) - Kill switch architecture and design
- [kill_switch.go](internal/mergequeue/kill_switch.go) - Implementation
- [kill_switch_test.go](internal/mergequeue/kill_switch_test.go) - Unit tests
- [killswitch_integration_test.go](internal/mergequeue/killswitch_integration_test.go) - Integration tests
