# Kill Switch Benchmarks - Quick Start Guide

This guide provides quick commands for running the kill switch performance benchmarks.

## Quick Commands

### Run all kill switch benchmarks
```bash
go test -bench=^Benchmark -v -run=^$ ./internal/mergequeue
```

### Run specific benchmark category

Kill single branches:
```bash
go test -bench=^BenchmarkKillFailed -v -run=^$ ./internal/mergequeue
```

Kill multiple branches:
```bash
go test -bench=^BenchmarkKillMultiple -v -run=^$ ./internal/mergequeue
```

Cascade kills (hierarchies):
```bash
go test -bench=^BenchmarkKillDependent -v -run=^$ ./internal/mergequeue
```

Protection mechanisms:
```bash
go test -bench=^BenchmarkKillProtection -v -run=^$ ./internal/mergequeue
```

Memory allocation:
```bash
go test -bench=^BenchmarkMemory -v -benchmem -run=^$ ./internal/mergequeue
```

Concurrent operations:
```bash
go test -bench=^BenchmarkConcurrent -v -run=^$ ./internal/mergequeue
```

### Run with memory reporting
```bash
go test -bench=^Benchmark -v -benchmem -run=^$ ./internal/mergequeue
```

### Run with custom duration
```bash
go test -bench=^Benchmark -benchtime=10s -v -run=^$ ./internal/mergequeue
```

### Run benchmarks multiple times (for statistical analysis)
```bash
go test -bench=^Benchmark -count=5 -v -run=^$ ./internal/mergequeue | tee results.txt
```

### Compare with previous results
```bash
# Generate baseline (first time)
go test -bench=^Benchmark -benchmem -count=5 -v -run=^$ ./internal/mergequeue > baseline.txt

# Generate current results
go test -bench=^Benchmark -benchmem -count=5 -v -run=^$ ./internal/mergequeue > current.txt

# Compare (requires golang.org/x/perf/cmd/benchstat)
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

## Benchmark Listing

| Benchmark | Purpose | Typical Time |
|-----------|---------|--------------|
| `BenchmarkKillFailedBranchWithTimeout_SingleBranch` | Baseline single kill | ~2 µs |
| `BenchmarkKillFailedBranchWithTimeout_IdempotentKill` | Fast-path idempotent kill | ~0.8 µs |
| `BenchmarkKillFailedBranchWithTimeout_KillWithTimeout` | Kill with timeout overhead | ~1.3 µs |
| `BenchmarkKillFailedBranchWithTimeout_MutexContention` | Kill with 1000 existing branches | ~1.7 µs |
| `BenchmarkKillMultipleBranchesSequential` | Sequential kill of 10 branches | ~12.6 µs |
| `BenchmarkKillMultipleBranchesParallel` | Parallel kill of 10 branches | ~17.2 µs |
| `BenchmarkKillDependentBranchesWithTimeout_ShallowHierarchy` | Cascade kill (3 children) | ~10.4 µs |
| `BenchmarkKillDependentBranchesWithTimeout_DeepHierarchy` | Cascade kill (10 levels) | ~28.2 µs |
| `BenchmarkKillDependentBranchesWithTimeout_WideHierarchy` | Cascade kill (50 children) | ~116 µs |
| `BenchmarkKillDependentBranchesWithTimeout_ComplexHierarchy` | Cascade kill (8 mixed nodes) | ~19.4 µs |
| `BenchmarkKillProtectionMechanism_MutexAcquisition` | Lock overhead | ~15 ns |
| `BenchmarkKillProtectionMechanism_TimeoutContextCreation` | Timeout context overhead | ~311 ns |
| `BenchmarkKillProtectionMechanism_ChannelSelect` | Channel select overhead | ~4 ns |
| `BenchmarkKillProtectionMechanism_IdempotencyCheck` | Idempotency check overhead | ~18 ns |
| `BenchmarkMemoryAllocation_BranchCreation` | Memory per branch | 24 B / 2 allocs |
| `BenchmarkMemoryAllocation_KilledBranchMetadata` | Memory per killed branch | 24 B / 2 allocs |
| `BenchmarkMemoryAllocation_HierarchyCreation` | Memory per branch with children | 264 B / 21 allocs |
| `BenchmarkConcurrentKillOperations` | 100 branches, 10 concurrent workers | ~119 µs |
| `BenchmarkCascadeKillTiming` | Binary tree (63 nodes) | ~143 µs |

## Performance Targets

| Operation | Target | Actual |
|-----------|--------|--------|
| Single kill latency | < 10 µs | ✓ 2 µs |
| Shallow cascade latency | < 50 µs | ✓ 10 µs |
| Deep cascade latency | < 100 µs | ✓ 28 µs |
| Wide cascade latency | < 500 µs | ✓ 116 µs |
| Lock overhead | < 100 ns | ✓ 15 ns |
| Timeout context overhead | < 500 ns | ✓ 311 ns |
| Allocation per branch | < 100 B | ✓ 24 B |

All benchmarks meet or exceed performance targets.

## Interpreting Results

### Time/op
- **ns**: nanoseconds (10^-9 seconds)
- **µs**: microseconds (10^-6 seconds)
- **ms**: milliseconds (10^-3 seconds)

For example: `1,966 ns/op` means each operation takes ~2 microseconds

### Ops
- Number of iterations completed in the benchmark period
- Higher is better (indicates faster operation)

### Memory Results (with `-benchmem`)
- **allocs/op**: Number of memory allocations per operation
- **B/op**: Bytes allocated per operation

## Profiling

For detailed performance analysis:

### CPU Profile
```bash
go test -bench=^BenchmarkKillDependentBranchesWithTimeout_DeepHierarchy$ -cpuprofile=cpu.prof -v ./internal/mergequeue
go tool pprof cpu.prof
```

### Memory Profile
```bash
go test -bench=^BenchmarkMemory -memprofile=mem.prof -v ./internal/mergequeue
go tool pprof mem.prof
```

### Trace (Go 1.11+)
```bash
go test -bench=^Benchmark -trace=trace.out -v ./internal/mergequeue
go tool trace trace.out
```

## Continuous Monitoring

Add to your CI/CD pipeline:

```bash
# Run benchmarks and save results
go test -bench=^Benchmark -benchmem -count=3 -v -run=^$ ./internal/mergequeue > benchmark_results.txt

# Store results for trend analysis
cat benchmark_results.txt >> benchmark_history.log
```

## More Information

- See [KILLSWITCH_BENCHMARKS.md](KILLSWITCH_BENCHMARKS.md) for detailed benchmark documentation
- See [KILLSWITCH.md](KILLSWITCH.md) for architecture and design documentation
- See [internal/mergequeue/kill_switch.go](internal/mergequeue/kill_switch.go) for implementation
