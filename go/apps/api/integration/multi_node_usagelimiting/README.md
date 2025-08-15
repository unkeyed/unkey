# Multi-Node Usage Limiting Integration Tests

This package contains comprehensive integration tests for the usage limiting system under multi-node scenarios. These tests verify both accuracy and performance characteristics.

## Test Structure

### Generated Tests (`generated/`)

Automatically generated tests that cover various combinations of:

- **Node counts**: 1, 3, 5, 9 nodes
- **Credit amounts**: 10, 100, 500, 1000 credits
- **Cost per request**: 1, 5, 10, 20 credits
- **Load factors**: 0.8x, 0.9x, 1.0x, 1.2x, 1.5x, 2.0x, 3.0x
- **Test durations**: 10-30 seconds

### Accuracy Tests (`accuracy_test.go`)

High-concurrency tests focused on credit counting accuracy:

- **Race condition detection**: 50-100 concurrent requests
- **Credit precision**: Verify exact credit consumption
- **Multi-node consistency**: Ensure distributed accuracy
- **Edge cases**: Single credit scenarios, high-cost operations

### Performance Tests (`performance_test.go`)

Benchmarks and latency measurements:

- **Throughput benchmarks**: Measure maximum RPS
- **Latency analysis**: P50, P95, P99 percentiles
- **Sustained load tests**: 10-second stress tests
- **Concurrency scaling**: 1-100 concurrent workers

## Running Tests

### Quick Integration Tests (Single Node)

```bash
go test -tags=integration ./apps/api/integration/multi_node_usagelimiting/generated/usagelimit_nodes01_*/
```

### Full Integration Suite

```bash
go test -tags=integration_long ./apps/api/integration/multi_node_usagelimiting/generated/*/
```

### Stress Tests (9+ Nodes)

```bash
go test -tags=stress ./apps/api/integration/multi_node_usagelimiting/generated/*/
```

### Accuracy Tests

```bash
go test -tags=integration -run TestUsageLimitAccuracy ./apps/api/integration/multi_node_usagelimiting/
```

### Performance Tests

```bash
go test -tags=integration -run TestUsageLimitLatency ./apps/api/integration/multi_node_usagelimiting/
go test -tags=integration -run TestUsageLimitThroughput ./apps/api/integration/multi_node_usagelimiting/
```

### Benchmarks

```bash
go test -tags=integration -bench=BenchmarkUsageLimitPerformance ./apps/api/integration/multi_node_usagelimiting/
```

## Test Scenarios

### Accuracy Verification

- **Credit precision**: Never exceed the credit limit
- **Race condition handling**: High concurrency doesn't cause over-spending
- **Multi-node consistency**: Distributed nodes maintain accuracy
- **Eventually consistent**: Redis and DB converge
- **Load balancing**: Traffic distributed across nodes

### Performance Expectations

- **Latency**: P95 < 200ms, P99 < 500ms
- **Throughput**: > 100 RPS sustained
- **Error rate**: < 1% under normal load

### Edge Cases Tested

- **Single credit exhaustion**: 1 credit, 100 concurrent requests
- **High-cost operations**: 20 credits per request
- **Extreme load**: 3x expected request rate
- **Node failures**: Graceful degradation (manual testing)

## Test Data Flow

1. **Setup**: Create API key with specific credit limit
2. **Load generation**: Concurrent requests across multiple nodes
3. **Real-time verification**: Check response accuracy
4. **ClickHouse validation**: Verify analytics data
5. **Database consistency**: Check eventual consistency
6. **Cleanup**: Automatic test resource cleanup

## Regenerating Tests

To add new test scenarios:

1. Edit `generate_tests/main.go`
2. Add new `TestCase` entries to `realisticCombinations` or `extremeEdgeCases`
3. Run the generator:
   ```bash
   cd generate_tests && go generate
   ```

## Key Metrics Tracked

- **Success rate**: Percentage of successful verifications
- **Credit accuracy**: Actual vs expected credit consumption
- **Latency distribution**: Response time percentiles
- **Node distribution**: Traffic spread across nodes
- **Error patterns**: Types and frequency of failures

These tests ensure the usage limiting system maintains both high performance and perfect accuracy under production-like loads across multiple nodes.
