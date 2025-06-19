# Billaged Development Guide

## Development Setup

### Prerequisites

- Go 1.24.3 or later
- Make
- Protocol Buffers compiler (for regenerating protos)
- Docker (for integration tests)
- Access to SPIFFE workload API (optional)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/billaged

# Install dependencies
go mod download

# Build the service
make build

# Run tests
make test
```

## Build Instructions

### Makefile Targets

**Makefile**: [`Makefile`](../../Makefile)

```bash
# Build binary
make build

# Run tests
make test

# Run integration tests
make test-integration

# Run benchmarks
make bench

# Install with systemd
make install

# Clean build artifacts
make clean

# Regenerate protobuf code
make proto
```

### Build Tags

The service supports build tags for different configurations:

```bash
# Build with specific features
go build -tags "debug,profiling" ./cmd/billaged
```

## Testing

### Unit Tests

Located alongside source files with `_test.go` suffix.

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/aggregator

# Verbose output
go test -v ./...
```

### Writing Tests

Example test structure:

```go
func TestAggregator_ProcessMetricsBatch(t *testing.T) {
    // Setup
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    agg := aggregator.NewAggregator(logger, 60*time.Second)
    
    // Test data
    metrics := []*billingv1.VMMetrics{
        {
            Timestamp:        timestamppb.Now(),
            CpuTimeNanos:     1000000000,
            MemoryUsageBytes: 1073741824,
        },
    }
    
    // Execute
    agg.ProcessMetricsBatch("vm-123", "customer-456", metrics)
    
    // Assert
    assert.Equal(t, 1, agg.GetActiveVMCount())
}
```

### Integration Tests

Test the service with real dependencies:

```bash
# Start dependencies
docker-compose -f test/docker-compose.yml up -d

# Run integration tests
go test -tags integration ./test/integration

# Cleanup
docker-compose -f test/docker-compose.yml down
```

### Benchmark Tests

Performance testing for critical paths:

```go
func BenchmarkAggregator_ProcessMetrics(b *testing.B) {
    agg := aggregator.NewAggregator(logger, 60*time.Second)
    metrics := generateTestMetrics(100)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        agg.ProcessMetricsBatch("vm-123", "customer-456", metrics)
    }
}
```

Run benchmarks:
```bash
go test -bench=. ./internal/aggregator
```

## Local Development

### Running Locally

Basic development setup:

```bash
# Set minimal configuration
export UNKEY_BILLAGED_PORT=8081
export UNKEY_BILLAGED_TLS_MODE=disabled
export UNKEY_BILLAGED_OTEL_ENABLED=false

# Run the service
go run ./cmd/billaged
```

### With Mock Dependencies

For testing without metald:

```bash
# Use the example client to send test data
cd contrib/example-client
go run main.go -action send-metrics
```

### With Docker Compose

Complete local environment:

```yaml
# docker-compose.yml
version: '3.8'
services:
  billaged:
    build: .
    ports:
      - "8081:8081"
      - "9465:9465"
    environment:
      - UNKEY_BILLAGED_OTEL_ENABLED=true
      - UNKEY_BILLAGED_OTEL_ENDPOINT=otel-collector:4318
    
  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4318:4318"
```

## Debugging

### Enable Debug Logging

Add to [`cmd/billaged/main.go`](../../cmd/billaged/main.go:87-90):

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
    AddSource: true,
}))
```

### Using Delve

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o billaged ./cmd/billaged

# Start with delve
dlv exec ./billaged

# Set breakpoints
(dlv) break internal/service/billing.go:42
(dlv) continue
```

### Profiling

Enable pprof endpoint:

```go
import _ "net/http/pprof"

// In main()
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Profile CPU usage:
```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

## Code Organization

### Directory Structure

```
billaged/
├── cmd/
│   └── billaged/
│       └── main.go           # Entry point
├── internal/
│   ├── aggregator/
│   │   └── aggregator.go     # Core aggregation logic
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── observability/
│   │   ├── interceptor.go    # OTEL interceptor
│   │   ├── metrics.go        # Prometheus metrics
│   │   └── otel.go          # OTEL setup
│   └── service/
│       └── billing.go        # Service implementation
├── proto/
│   └── billing/v1/
│       └── billing.proto     # API definitions
├── gen/                      # Generated code
├── contrib/
│   ├── systemd/             # Systemd units
│   └── grafana-dashboards/  # Monitoring
└── Makefile
```

### Code Style

Follow standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Check for issues
go vet ./...
```

## API Development

### Modifying the API

1. Edit [`proto/billing/v1/billing.proto`](../../proto/billing/v1/billing.proto)
2. Regenerate code:
   ```bash
   make proto
   ```
3. Implement new methods in [`internal/service/billing.go`](../../internal/service/billing.go)

### Adding New Metrics

1. Define metric in [`internal/observability/metrics.go`](../../internal/observability/metrics.go):
   ```go
   vmLifecycleDuration, err := meter.Float64Histogram(
       "billaged_vm_lifecycle_duration_seconds",
       metric.WithDescription("VM lifetime duration"),
   )
   ```

2. Record metric in service:
   ```go
   m.vmLifecycleDuration.Record(ctx, duration.Seconds(),
       metric.WithAttributes(
           attribute.String("customer_id", customerID),
       ),
   )
   ```

## Dependency Management

### Adding Dependencies

```bash
# Add a new dependency
go get github.com/some/package@latest

# Update go.mod and go.sum
go mod tidy

# Verify dependencies
go mod verify
```

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/prometheus/client_golang

# Clean up
go mod tidy
```

## Contributing

### Pre-commit Checks

Run before committing:

```bash
# Format code
go fmt ./...

# Run tests
go test ./...

# Check for common issues
golangci-lint run

# Verify build
make build
```

### Commit Message Format

Follow conventional commits:

```
feat: add VM lifecycle duration metric
fix: handle counter reset in aggregator
docs: update API documentation
test: add aggregator benchmark tests
refactor: extract score calculation logic
```

## Troubleshooting Development Issues

### Module Issues

```bash
# Clear module cache
go clean -modcache

# Download dependencies again
go mod download

# Verify module integrity
go mod verify
```

### Proto Generation Issues

```bash
# Install protoc
brew install protobuf  # macOS
apt install protobuf-compiler  # Ubuntu

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

### Build Issues

```bash
# Clean build cache
go clean -cache

# Verbose build
go build -v ./cmd/billaged

# Check for errors
go build -x ./cmd/billaged 2>&1 | grep -i error
```

## Performance Optimization

### Profiling Checklist

1. **CPU Profile**: Identify hot paths
2. **Memory Profile**: Find allocations
3. **Goroutine Profile**: Check for leaks
4. **Block Profile**: Find contention

### Common Optimizations

1. **Reduce Allocations**:
   ```go
   // Reuse slices
   metrics = metrics[:0]
   ```

2. **Batch Operations**:
   ```go
   // Process in batches
   for i := 0; i < len(metrics); i += batchSize {
       end := min(i+batchSize, len(metrics))
       processBatch(metrics[i:end])
   }
   ```

3. **Concurrent Processing**:
   ```go
   // Use worker pool
   for i := 0; i < workers; i++ {
       go worker(jobs, results)
   }
   ```

## Release Process

### Version Management

Update version in:
1. [`cmd/billaged/main.go`](../../cmd/billaged/main.go:31) - `version` variable
2. [`internal/config/config.go`](../../internal/config/config.go:158) - Default service version
3. [`Makefile`](../../Makefile) - VERSION variable

### Release Checklist

1. Run all tests
2. Update CHANGELOG.md
3. Tag release: `git tag -a v0.1.1 -m "Release v0.1.1"`
4. Build release binaries
5. Update documentation