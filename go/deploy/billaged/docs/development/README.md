# Billaged Development Setup

This guide covers building, testing, and contributing to the billaged service.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Development Environment](#development-environment)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Code Generation](#code-generation)
- [Contributing](#contributing)
- [Debugging](#debugging)

## Prerequisites

### Required Tools

- **Go**: 1.22.0 or higher ([go.mod:3](../../go.mod:3))
- **Make**: For build automation
- **protoc**: Protocol buffer compiler (v3.19+)
- **buf**: For protobuf management
- **golangci-lint**: For code linting

### Optional Tools

- **Docker**: For containerized builds
- **SPIRE**: For testing mTLS locally
- **Prometheus**: For metrics testing
- **grpcurl**: For API testing

### Tool Installation

```bash
# Install Go (via official installer or package manager)
# https://golang.org/dl/

# Install protoc
brew install protobuf  # macOS
apt install protobuf-compiler  # Ubuntu/Debian

# Install buf
curl -sSL https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64 \
  -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

## Development Environment

### Project Structure

```
billaged/
├── cmd/
│   └── billaged/
│       └── main.go              # Application entry point
├── internal/
│   ├── aggregator/              # Core aggregation logic
│   │   └── aggregator.go
│   ├── config/                  # Configuration management
│   │   └── config.go
│   ├── observability/           # Metrics and tracing
│   │   ├── interceptor.go
│   │   ├── metrics.go
│   │   └── otel.go
│   └── service/                 # Service implementation
│       └── billing.go
├── proto/
│   └── billing/v1/              # Protocol buffer definitions
│       └── billing.proto
├── gen/                         # Generated code (do not edit)
│   └── billing/v1/
├── contrib/
│   ├── systemd/                 # Systemd unit files
│   └── grafana-dashboards/      # Monitoring dashboards
├── Makefile                     # Build automation
├── go.mod                       # Go module definition
└── go.sum                       # Dependency checksums
```

### Environment Setup

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/billaged

# Download dependencies
go mod download

# Verify setup
go mod verify
```

### IDE Configuration

#### VS Code

`.vscode/settings.json`:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.testFlags": ["-v"],
  "go.buildTags": "",
  "go.generateTestsFlags": ["-exported"]
}
```

#### GoLand

1. Enable Go modules support
2. Set GOPATH to module mode
3. Configure golangci-lint as external tool

## Building from Source

### Make Targets

**Makefile**: [`Makefile:1-63`](../../Makefile:1-63)

```bash
# Build binary
make build

# Run tests
make test

# Run linter
make lint

# Generate code
make generate

# Install systemd service
make install

# Clean build artifacts
make clean

# Run all CI checks
make ci
```

### Manual Build

```bash
# Build for current platform
go build -o billaged ./cmd/billaged

# Build with version info
go build -ldflags "-X main.version=v0.1.0" -o billaged ./cmd/billaged

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o billaged-linux-amd64 ./cmd/billaged

# Build with race detector (development only)
go build -race -o billaged-race ./cmd/billaged
```

### Docker Build

```dockerfile
# Dockerfile example
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o billaged ./cmd/billaged

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/billaged /billaged
ENTRYPOINT ["/billaged"]
```

```bash
# Build Docker image
docker build -t billaged:dev .

# Run with environment variables
docker run -e UNKEY_BILLAGED_TLS_MODE=disabled billaged:dev
```

## Testing

### Unit Tests

**Test files**: [`*_test.go` pattern](../../internal/)

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./internal/aggregator

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestAggregator_ProcessMetricsBatch ./internal/aggregator
```

### Integration Tests

```bash
# Start test dependencies
docker-compose -f test/docker-compose.yml up -d

# Run integration tests
go test -tags=integration ./test/integration

# Cleanup
docker-compose -f test/docker-compose.yml down
```

### Test Patterns

#### Table-Driven Tests

```go
func TestAggregator_CalculateDelta(t *testing.T) {
    tests := []struct {
        name     string
        current  int64
        previous int64
        want     int64
    }{
        {"normal increment", 100, 50, 50},
        {"counter reset", 10, 100, 10},
        {"first value", 100, 0, 100},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := calculateDelta(tt.current, tt.previous)
            if got != tt.want {
                t.Errorf("got %d, want %d", got, tt.want)
            }
        })
    }
}
```

#### Mock Testing

```go
// Mock billing client for testing
type mockBillingClient struct {
    recordedMetrics []VMMetrics
}

func (m *mockBillingClient) SendMetricsBatch(ctx context.Context, batch []VMMetrics) error {
    m.recordedMetrics = append(m.recordedMetrics, batch...)
    return nil
}
```

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./internal/aggregator

# Run with memory profiling
go test -bench=. -benchmem ./internal/aggregator

# Compare benchmarks
go test -bench=. ./internal/aggregator > old.txt
# make changes
go test -bench=. ./internal/aggregator > new.txt
benchstat old.txt new.txt
```

## Code Generation

### Protocol Buffers

**Proto files**: [`proto/billing/v1/`](../../proto/billing/v1/)

```bash
# Generate Go code from proto files
make generate

# Manual generation
buf generate

# Verify generated code is up to date
git diff --exit-code gen/
```

### buf Configuration

`buf.gen.yaml`:
```yaml
version: v1
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - plugin: buf.build/connectrpc/go
    out: gen
    opt: paths=source_relative
```

## Contributing

### Code Style

The project uses standard Go formatting and conventions:

```bash
# Format code
gofmt -w .

# Run linter
golangci-lint run

# Fix linter issues
golangci-lint run --fix
```

### Commit Guidelines

Follow conventional commits:

```
feat: add new aggregation algorithm
fix: handle counter resets properly
docs: update API documentation
test: add benchmarks for aggregator
refactor: simplify metric processing
```

### Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run CI checks: `make ci`
5. Commit with clear message
6. Push branch and create PR

### Code Review Checklist

- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated
- [ ] Benchmarks if performance-critical
- [ ] No sensitive data in logs
- [ ] Error handling follows patterns

## Debugging

### Local Development

```bash
# Run with debug logging
go run ./cmd/billaged

# Run with specific config
UNKEY_BILLAGED_TLS_MODE=disabled \
UNKEY_BILLAGED_AGGREGATION_INTERVAL=10s \
go run ./cmd/billaged
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/billaged

# Debug with arguments
dlv debug ./cmd/billaged -- --config=debug.yaml

# Attach to running process
dlv attach $(pgrep billaged)
```

### VS Code Debug Configuration

`.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug billaged",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/billaged",
      "env": {
        "UNKEY_BILLAGED_TLS_MODE": "disabled",
        "UNKEY_BILLAGED_LOG_LEVEL": "debug"
      }
    }
  ]
}
```

### Performance Profiling

```bash
# CPU profiling
go run -cpuprofile=cpu.prof ./cmd/billaged

# Memory profiling
go run -memprofile=mem.prof ./cmd/billaged

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Generate flame graph
go tool pprof -http=:8080 cpu.prof
```

### Testing with grpcurl

```bash
# List services
grpcurl -plaintext localhost:8081 list

# Describe service
grpcurl -plaintext localhost:8081 describe billing.v1.BillingService

# Send test request
grpcurl -plaintext -d '{
  "vm_id": "test-vm",
  "customer_id": "test-customer",
  "metrics": [{
    "timestamp": "2024-01-01T12:00:00Z",
    "cpu_time_nanos": 1000000000,
    "memory_usage_bytes": 1073741824
  }]
}' localhost:8081 billing.v1.BillingService/SendMetricsBatch
```

## Common Development Tasks

### Adding a New RPC Method

1. Update proto file:
   ```protobuf
   service BillingService {
     // Existing methods...
     rpc NewMethod(NewMethodRequest) returns (NewMethodResponse);
   }
   ```

2. Generate code:
   ```bash
   make generate
   ```

3. Implement handler in [`internal/service/billing.go`](../../internal/service/billing.go):
   ```go
   func (s *BillingServiceServer) NewMethod(
       ctx context.Context,
       req *connect.Request[billingv1.NewMethodRequest],
   ) (*connect.Response[billingv1.NewMethodResponse], error) {
       // Implementation
   }
   ```

4. Add tests:
   ```go
   func TestBillingService_NewMethod(t *testing.T) {
       // Test implementation
   }
   ```

### Adding Metrics

1. Define metric in [`internal/observability/metrics.go`](../../internal/observability/metrics.go):
   ```go
   newMetric, err := meter.Float64Counter(
       "billaged_new_metric_total",
       metric.WithDescription("Description"),
   )
   ```

2. Record metric in service:
   ```go
   s.metrics.NewMetric.Add(ctx, 1, 
       metric.WithAttributes(
           attribute.String("key", "value"),
       ),
   )
   ```

### Updating Dependencies

```bash
# Update specific dependency
go get -u github.com/connectrpc/connect-go

# Update all dependencies
go get -u ./...

# Tidy module
go mod tidy

# Verify
go mod verify
```

## Troubleshooting Development Issues

### Common Problems

1. **Generated code out of sync**
   ```bash
   make generate
   git add gen/
   ```

2. **Import errors**
   ```bash
   go mod tidy
   go mod download
   ```

3. **Linter failures**
   ```bash
   golangci-lint run --fix
   ```

4. **Test failures**
   ```bash
   go test -v ./... -short
   ```

### Getting Help

- Check existing issues on GitHub
- Review AIDEV comments in code
- Consult Go documentation
- Ask in project discussions

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
1. [`cmd/billaged/main.go:31`](../../cmd/billaged/main.go:31) - `version` variable
2. [`internal/config/config.go:25`](../../internal/config/config.go:25) - Default service version
3. [`Makefile`](../../Makefile) - VERSION variable

### Release Checklist

1. Run all tests: `make ci`
2. Update CHANGELOG.md
3. Tag release: `git tag -a v0.1.1 -m "Release v0.1.1"`
4. Build release binaries
5. Update documentation