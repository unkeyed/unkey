# Logdrain Performance Testing

This directory contains performance testing scripts for the logdrain service to validate throughput, latency, and cursor advancement under load.

## Test Scenarios

### 1. Cursor Advancement Performance (`cursor_perf.go`)
- Tests cursor ordering and advancement speed
- Validates no race conditions under concurrent load
- Measures query latency for different group sizes

### 2. Provider Sink Throughput (`sink_throughput.go`) 
- Tests Axiom sink performance
- Measures records/second delivery rates
- Validates rate limiting and back-pressure

### 3. End-to-End Pipeline (`e2e_pipeline.sh`)
- Full logdrain pipeline with mock ClickHouse data
- Tests coordinator → sink → provider flow
- Measures overall system throughput

### 4. Group Fan-out Scalability (`group_fanout.go`)
- Tests read amplification limits (one query → many drains)
- Validates MaxGroupsPerShard enforcement
- Measures ClickHouse query impact

## Running Tests

```bash
# Setup test environment
make logdrain-perf-setup

# Run cursor advancement tests
go run benchmarks/logdrain/cursor_perf.go

# Run sink throughput tests
go run benchmarks/logdrain/sink_throughput.go

# Run full pipeline test
./benchmarks/logdrain/e2e_pipeline.sh

# Run scalability tests  
go run benchmarks/logdrain/group_fanout.go
```

## Metrics to Monitor

- **Cursor advancement latency** (target: <10ms p95)
- **Records/second throughput** (target: 10k+ records/sec)
- **ClickHouse query latency** (target: <500ms p95)
- **Group processing time** (target: <5s per tick)
- **Memory usage** (target: <500MB steady state)

## Expected Results

Based on the hardened architecture:
- Zero cursor ordering race conditions
- Linear throughput scaling with shard count
- Graceful degradation under provider back-pressure
- Memory usage bounded by group limits
