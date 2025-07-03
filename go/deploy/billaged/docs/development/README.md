# Billaged Development Setup

This guide covers building, testing, and developing the billaged service locally.

## Development Environment

### Prerequisites

#### Required Software
- **Go 1.24.4+**: Primary development language ([go.mod](../../go.mod#L3))
- **Protocol Buffers**: `protoc` compiler for gRPC/ConnectRPC code generation
- **Buf**: Protocol buffer linting and generation tool
- **Make**: Build automation and development tasks
- **Docker**: Container runtime for integration testing (optional)

#### Development Dependencies

**Go Module**: [go.mod](../../go.mod)

```bash
# Core dependencies
connectrpc.com/connect v1.18.1                    # ConnectRPC framework
github.com/prometheus/client_golang v1.22.0       # Metrics collection
go.opentelemetry.io/otel v1.37.0                 # Observability framework
google.golang.org/protobuf v1.36.6               # Protocol buffer runtime

# Unkey internal packages
github.com/unkeyed/unkey/go/deploy/pkg/health     # Health check utilities
github.com/unkeyed/unkey/go/deploy/pkg/tls        # TLS/SPIFFE integration
```

### Environment Setup

#### Go Environment

```bash
# Verify Go installation
go version  # Should be 1.24.4+

# Set Go environment
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Clone repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/billaged
```

#### Protocol Buffer Setup

```bash
# Install buf (recommended)
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" \
  -o "/usr/local/bin/buf"
chmod +x /usr/local/bin/buf

# Alternative: Install protoc directly
sudo apt-get install protobuf-compiler  # Ubuntu/Debian
brew install protobuf                   # macOS
```

## Building from Source

### Build System

**Build Configuration**: [Makefile](../../Makefile)

#### Standard Build

```bash
# Clean build
make clean
make build

# Output: build/billaged binary
./build/billaged --version
```

#### Development Build

```bash
# Build with debug symbols
make build-debug

# Build with race detection
make build-race

# Cross-compilation for Linux
GOOS=linux GOARCH=amd64 make build
```

#### Protocol Buffer Generation

```bash
# Generate Go code from proto files
make proto-gen

# Lint protocol buffer files
make proto-lint

# Clean generated files
make proto-clean
```

**Generated Files**: [gen/](../../gen/)
- `billing/v1/billing.pb.go` - Protocol buffer types
- `billing/v1/billingv1connect/billing.connect.go` - ConnectRPC service stubs

### Build Targets

#### Available Make Targets

```bash
make help                    # Show all available targets
make build                   # Build production binary
make test                    # Run unit tests
make test-integration        # Run integration tests
make test-coverage           # Generate test coverage report
make install                 # Install binary and systemd unit
make clean                   # Clean build artifacts
make fmt                     # Format Go code
make lint                    # Run Go linting
make proto-gen               # Generate protobuf code
make proto-lint              # Lint protobuf files
```

## Local Development

### Development Configuration

#### Minimal Development Setup

```bash
# Development environment variables
export UNKEY_BILLAGED_PORT=8081
export UNKEY_BILLAGED_ADDRESS=127.0.0.1         # Localhost only
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=10s  # Faster feedback
export UNKEY_BILLAGED_TLS_MODE=disabled         # No SPIFFE required
export UNKEY_BILLAGED_OTEL_ENABLED=false        # Simplified observability
```

#### Full Development Setup

```bash
# Full-featured development environment
export UNKEY_BILLAGED_PORT=8081
export UNKEY_BILLAGED_ADDRESS=0.0.0.0
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s
export UNKEY_BILLAGED_TLS_MODE=spiffe
export UNKEY_BILLAGED_SPIFFE_SOCKET=/tmp/spire/agent.sock
export UNKEY_BILLAGED_OTEL_ENABLED=true
export UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED=true
export UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT=9465
```

### Running Locally

#### Development Server

```bash
# Run with development configuration
make run-dev

# Run with custom configuration
go run cmd/billaged/main.go

# Run with specific log level
UNKEY_BILLAGED_LOG_LEVEL=debug go run cmd/billaged/main.go
```

#### Using Air for Live Reload

```bash
# Install air for live reloading
go install github.com/cosmtrek/air@latest

# Create .air.toml configuration
cat > .air.toml << EOF
root = "."
cmd = "go run cmd/billaged/main.go"
include_ext = ["go"]
exclude_dir = ["build", "gen", "docs"]
EOF

# Start development server with live reload
air
```

### Client Development

#### CLI Tool Development

**CLI Source**: [cmd/billaged-cli/main.go](../../cmd/billaged-cli/main.go)

```bash
# Build CLI tool
cd cmd/billaged-cli
go build -o billaged-cli

# Test CLI commands
./billaged-cli -server=http://localhost:8081 heartbeat
./billaged-cli -server=http://localhost:8081 send-metrics
./billaged-cli -server=http://localhost:8081 notify-started vm-test-123
```

#### Client Library Development

**Client Library**: [client/](../../client/)

```go
// Example client usage for development
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/unkeyed/unkey/go/deploy/billaged/client"
    billingv1 "github.com/unkeyed/unkey/go/deploy/billaged/gen/billing/v1"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
    ctx := context.Background()
    
    // Development client configuration
    config := client.Config{
        ServerAddress: "http://localhost:8081",
        UserID:       "dev-user-123",
        TenantID:     "dev-tenant-456", 
        TLSMode:      "disabled",
        Timeout:      10 * time.Second,
    }
    
    client, err := client.New(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Send test metrics
    metrics := []*billingv1.VMMetrics{{
        Timestamp:        timestamppb.Now(),
        CpuTimeNanos:     1000000000,
        MemoryUsageBytes: 512 * 1024 * 1024,
        DiskReadBytes:    1024 * 1024,
        DiskWriteBytes:   512 * 1024,
        NetworkRxBytes:   2048,
        NetworkTxBytes:   1024,
    }}
    
    resp, err := client.SendMetricsBatch(ctx, &client.SendMetricsBatchRequest{
        VmID:       "dev-vm-123",
        CustomerID: "dev-tenant-456",
        Metrics:    metrics,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response: %+v", resp)
}
```

## Testing

### Unit Testing

#### Test Structure

**Test Files**: Located alongside source files with `_test.go` suffix

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/service/...
go test ./internal/aggregator/...
go test ./internal/config/...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

#### Test Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
go tool cover -html=coverage.out

# Coverage by package
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out
```

### Integration Testing

#### Service Integration Tests

**Test Location**: [internal/service/](../../internal/service/)

```go
// Example integration test structure
func TestBillingServiceIntegration(t *testing.T) {
    // Setup test aggregator
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    agg := aggregator.NewAggregator(logger, 5*time.Second)
    
    // Create service with test metrics
    service := service.NewBillingService(logger, agg, nil)
    
    // Test metrics processing
    req := &billingv1.SendMetricsBatchRequest{
        VmId:       "test-vm-123",
        CustomerId: "test-customer",
        Metrics:    generateTestMetrics(),
    }
    
    resp, err := service.SendMetricsBatch(context.Background(), connect.NewRequest(req))
    require.NoError(t, err)
    assert.True(t, resp.Msg.Success)
}
```

#### End-to-End Testing

```bash
# Start development server
UNKEY_BILLAGED_TLS_MODE=disabled go run cmd/billaged/main.go &
SERVER_PID=$!

# Run integration tests
go test -tags=integration ./test/integration/...

# Cleanup
kill $SERVER_PID
```

### Mock Testing

#### Aggregator Mock Testing

**Test Implementation**: [internal/aggregator/aggregator_test.go](../../internal/aggregator/)

```go
func TestAggregatorResourceScoring(t *testing.T) {
    tests := []struct {
        name           string
        metrics        []*billingv1.VMMetrics
        expectedScore  float64
    }{
        {
            name: "cpu_heavy_workload",
            metrics: generateCPUHeavyMetrics(),
            expectedScore: 3.5, // Expected resource score
        },
        {
            name: "memory_heavy_workload", 
            metrics: generateMemoryHeavyMetrics(),
            expectedScore: 2.8,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agg := aggregator.NewAggregator(logger, time.Minute)
            agg.ProcessMetricsBatch("test-vm", "test-customer", tt.metrics)
            
            // Verify resource score calculation
            summary := agg.GenerateUsageSummary("test-vm")
            assert.InDelta(t, tt.expectedScore, summary.ResourceScore, 0.1)
        })
    }
}
```

### Performance Testing

#### Benchmark Tests

```go
// Benchmark aggregation performance
func BenchmarkAggregatorProcessing(b *testing.B) {
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    agg := aggregator.NewAggregator(logger, time.Minute)
    metrics := generateLargeMetricsBatch(100) // 100 metrics
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        agg.ProcessMetricsBatch("bench-vm", "bench-customer", metrics)
    }
}
```

#### Load Testing

```bash
# Install load testing tool
go install github.com/rakyll/hey@latest

# Basic load test
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"vm_id":"load-test","customer_id":"test","metrics":[]}' \
  http://localhost:8081/billing.v1.BillingService/SendMetricsBatch

# Sustained load test
hey -n 10000 -c 50 -q 100 \
  -H "Content-Type: application/json" \
  -d @test-metrics.json \
  http://localhost:8081/billing.v1.BillingService/SendMetricsBatch
```

## Debugging

### Debugging Tools

#### Go Debugging with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug main binary
dlv debug cmd/billaged/main.go

# Debug with arguments
dlv debug cmd/billaged/main.go -- --help

# Attach to running process
dlv attach $(pgrep billaged)
```

#### Debug Configuration

```bash
# Enable debug logging
export UNKEY_BILLAGED_LOG_LEVEL=debug

# Enable Go runtime debugging
export GODEBUG=gctrace=1           # GC debugging
export GODEBUG=schedtrace=1000     # Scheduler debugging
```

### Development Observability

#### Local Prometheus Setup

```bash
# Start Prometheus with development configuration
cat > prometheus-dev.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'billaged-dev'
    static_configs:
      - targets: ['localhost:9465']
    scrape_interval: 5s
EOF

# Run Prometheus
prometheus --config.file=prometheus-dev.yml --web.listen-address=localhost:9090
```

#### Local Grafana Setup

```bash
# Start Grafana in development mode
docker run -d -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  grafana/grafana

# Import development dashboard
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @contrib/grafana-dashboards/development.json
```

### Local SPIFFE Setup

#### Development SPIRE Configuration

```bash
# Start SPIRE server for development
spire-server run -config spire/environments/development/server.conf &

# Start SPIRE agent
spire-agent run -config spire/environments/development/agent.conf &

# Register billaged workload
spire-server entry create \
  -spiffeID spiffe://dev.unkey.io/billaged \
  -parentID spiffe://dev.unkey.io/agent \
  -selector unix:uid:$(id -u) \
  -selector unix:gid:$(id -g)

# Verify registration
spire-server entry show
```

## Code Quality

### Linting and Formatting

#### Go Code Quality

```bash
# Format code
make fmt
gofmt -w .
goimports -w .

# Lint code
make lint
golangci-lint run

# Vet code
go vet ./...

# Check for security issues
gosec ./...
```

#### Protocol Buffer Linting

```bash
# Lint protobuf files
make proto-lint
buf lint

# Format protobuf files
buf format -w

# Check breaking changes
buf breaking --against .git#HEAD~1
```

### Pre-commit Hooks

#### Git Hooks Setup

```bash
# Install pre-commit hooks
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Format code
make fmt

# Run linting
make lint

# Run tests
make test

echo "Pre-commit checks passed!"
EOF

chmod +x .git/hooks/pre-commit
```

### Documentation

#### Code Documentation

```bash
# Generate Go documentation
godoc -http=:6060

# View documentation in browser
open http://localhost:6060/pkg/github.com/unkeyed/unkey/go/deploy/billaged/
```

#### API Documentation

```bash
# Generate protobuf documentation
buf generate --template buf.gen.doc.yaml

# View generated documentation
open docs/api/
```

## Contributing

### Development Workflow

#### Standard Development Flow

1. **Feature Branch**: Create feature branch from main
2. **Development**: Implement feature with tests
3. **Testing**: Run full test suite including integration tests
4. **Documentation**: Update relevant documentation
5. **Review**: Submit pull request for code review
6. **Integration**: Merge after approval and CI success

#### Commit Message Format

```
type(scope): description

body

footer
```

**Examples**:
```
feat(aggregator): add resource score weighting configuration
fix(service): handle empty metrics batch gracefully  
docs(api): update SendMetricsBatch example
test(client): add integration test for SPIFFE authentication
```

### Development Best Practices

#### Code Organization

- **Package Structure**: Follow Go package conventions with clear separation of concerns
- **Error Handling**: Use structured errors with context for debugging
- **Logging**: Use structured logging with appropriate levels and context
- **Testing**: Write tests for all public APIs and critical business logic
- **Documentation**: Document all public APIs and complex algorithms

#### Performance Considerations

- **Memory Efficiency**: Minimize allocations in hot paths
- **Concurrency**: Use appropriate synchronization for shared data structures
- **Resource Cleanup**: Ensure proper cleanup of resources and goroutines
- **Metrics**: Add appropriate metrics for monitoring performance

#### Security Practices

- **Input Validation**: Validate all external inputs thoroughly
- **Authentication**: Enforce SPIFFE authentication in production code paths
- **Tenant Isolation**: Ensure customer data isolation at all levels
- **Secrets Management**: Never commit secrets or certificates to source control