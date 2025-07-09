# VM Cleanup Performance Benchmarks

This document describes the performance benchmarks for the `performVMCleanup()` method and how to interpret the results.

## Overview

The VM cleanup process is critical for preventing resource leaks when VM creation fails after backend operations succeed. These benchmarks test various scenarios to ensure the cleanup mechanism performs well under load.

## Running Benchmarks

### Basic Benchmark Run

```bash
# Run all cleanup benchmarks
go test -bench=BenchmarkCleanup -benchmem ./internal/service/

# Run specific benchmark
go test -bench=BenchmarkCleanupSuccess -benchmem ./internal/service/

# Run with verbose output
go test -bench=BenchmarkCleanup -benchmem -v ./internal/service/
```

### Extended Benchmark Run

```bash
# Run for longer duration to get stable results
go test -bench=BenchmarkCleanup -benchtime=10s -benchmem ./internal/service/

# Run with CPU profiling
go test -bench=BenchmarkCleanupConcurrent -cpuprofile=cleanup.prof ./internal/service/

# Run with memory profiling
go test -bench=BenchmarkCleanupMemoryUsage -memprofile=cleanup_mem.prof ./internal/service/
```

## Benchmark Scenarios

### 1. BenchmarkCleanupSuccess
**Purpose**: Tests optimal performance with fast, successful backend operations.
**Conditions**: 10ms backend latency, 0% failure rate
**Key Metrics**:
- Operations per second
- Memory allocations per operation
- Backend calls (should equal number of operations)

### 2. BenchmarkCleanupWithRetries
**Purpose**: Tests performance when retries are frequently needed.
**Conditions**: 5ms backend latency, 40% failure rate
**Key Metrics**:
- Operations per second (lower than success case)
- Backend calls (should be 1.6x operations due to retries)
- Memory allocations (higher due to retry logic)

### 3. BenchmarkCleanupConcurrent
**Purpose**: Tests scalability with concurrent cleanup operations.
**Conditions**: Variable concurrency levels (1, 10, 50, 100, 200)
**Key Metrics**:
- Throughput scaling with concurrency
- Maximum concurrent backend calls
- Memory allocation patterns

### 4. BenchmarkCleanupSlowBackend
**Purpose**: Tests performance with slow backend responses.
**Conditions**: Variable latencies (50ms to 1s)
**Key Metrics**:
- Impact of backend latency on overall performance
- Context timeout behavior
- Resource usage during waiting

### 5. BenchmarkCleanupContextCancellation
**Purpose**: Tests grace period context functionality.
**Conditions**: Original context cancelled before operation completes
**Key Metrics**:
- Success rate (should be 100% due to grace period)
- Grace period effectiveness
- Resource cleanup behavior

### 6. BenchmarkCleanupMemoryUsage
**Purpose**: Measures memory allocation patterns in detail.
**Conditions**: Fast operations with some failures
**Key Metrics**:
- Bytes allocated per operation
- Number of allocations per operation
- Memory allocation efficiency

### 7. BenchmarkCleanupStressTest
**Purpose**: Tests burst scenarios with many concurrent cleanups.
**Conditions**: Burst sizes from 10 to 500 concurrent operations
**Key Metrics**:
- Burst completion time
- Maximum concurrent backend calls
- System resource usage

### 8. BenchmarkCleanupFailureRecovery
**Purpose**: Tests behavior under total backend failure.
**Conditions**: 100% backend failure rate
**Key Metrics**:
- Retry behavior (should see exactly 3x backend calls)
- Failure detection speed
- Resource cleanup after failures

## Interpreting Results

### Sample Output Explanation

```
BenchmarkCleanupSuccess-8         	    1000	   1205834 ns/op	     328 B/op	      12 allocs/op
	backend_calls: 1000
	max_concurrent: 8
```

**Breakdown**:
- `1000`: Number of iterations completed
- `1205834 ns/op`: Average time per operation (1.2ms)
- `328 B/op`: Bytes allocated per operation
- `12 allocs/op`: Number of memory allocations per operation
- `backend_calls: 1000`: Total backend calls made
- `max_concurrent: 8`: Maximum concurrent backend operations

### Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| **Successful Cleanup** | < 50ms/op | Fast cleanup prevents request delays |
| **With Retries** | < 150ms/op | 3 retries with backoff should complete quickly |
| **Memory Usage** | < 1KB/op | Low allocation prevents GC pressure |
| **Concurrent Scaling** | Linear to 100 ops | Should scale well on multi-core systems |
| **Failure Recovery** | < 5s total | Quick failure detection and reporting |

### Warning Signs

🚨 **Performance Issues to Watch For**:

1. **High Memory Allocation**
   ```
   10000 B/op	    500 allocs/op
   ```
   - Indicates potential memory leaks or inefficient allocation patterns

2. **Poor Concurrency Scaling**
   ```
   BenchmarkCleanupConcurrent/concurrency-1-8    1000  1000000 ns/op
   BenchmarkCleanupConcurrent/concurrency-100-8    10  100000000 ns/op  # 100x slower!
   ```
   - Should scale roughly linearly with concurrency

3. **Excessive Backend Calls**
   ```
   backend_calls: 5000  # For 1000 operations - indicates retry storms
   ```

4. **Context Grace Period Failures**
   ```
   BenchmarkCleanupContextCancellation: 50% success rate
   ```
   - Should be nearly 100% due to grace period context

## Performance Analysis Tools

### CPU Profiling

```bash
# Generate CPU profile
go test -bench=BenchmarkCleanupConcurrent -cpuprofile=cpu.prof ./internal/service/

# Analyze with pprof
go tool pprof cpu.prof
(pprof) top10
(pprof) web
```

### Memory Profiling

```bash
# Generate memory profile
go test -bench=BenchmarkCleanupMemoryUsage -memprofile=mem.prof ./internal/service/

# Analyze memory usage
go tool pprof mem.prof
(pprof) top10
(pprof) list performVMCleanup
```

### Trace Analysis

```bash
# Generate execution trace
go test -bench=BenchmarkCleanupStressTest -trace=trace.out ./internal/service/

# View trace
go tool trace trace.out
```

## Production Monitoring

Based on benchmark results, configure production monitoring:

### Metrics to Track

```yaml
# Prometheus metrics
- metald_vm_cleanup_duration_seconds
- metald_vm_cleanup_attempts_total
- metald_vm_cleanup_failures_total
- metald_vm_cleanup_concurrent_operations

# Alerts
- alert: VMCleanupSlow
  expr: histogram_quantile(0.95, metald_vm_cleanup_duration_seconds) > 0.1
  for: 5m
  
- alert: VMCleanupHighFailureRate
  expr: rate(metald_vm_cleanup_failures_total[5m]) > 0.05
  for: 10m
```

### Performance Baselines

Use benchmark results to establish baselines:

```bash
# Save baseline performance
go test -bench=BenchmarkCleanup -benchmem ./internal/service/ > baseline.txt

# Compare against baseline
go test -bench=BenchmarkCleanup -benchmem ./internal/service/ > current.txt
benchcmp baseline.txt current.txt
```

## Continuous Integration

Add performance regression testing:

```yaml
# .github/workflows/performance.yml
- name: Run Cleanup Benchmarks
  run: |
    go test -bench=BenchmarkCleanup -benchmem ./internal/service/ > bench.txt
    # Store results and compare against previous runs
```

## Optimization Guidelines

Based on benchmark results:

1. **If memory usage is high**: Look for unnecessary allocations in retry logic
2. **If concurrency doesn't scale**: Check for lock contention or blocking operations
3. **If retries are excessive**: Tune failure detection or backend timeouts
4. **If grace period fails**: Increase timeout or optimize critical path

These benchmarks provide comprehensive coverage of the cleanup performance characteristics and help ensure the system remains stable under various load conditions.