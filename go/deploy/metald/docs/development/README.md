# Metald Development Guide

This guide covers building, testing, and developing metald.

## Table of Contents

- [Development Environment](#development-environment)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Local Development](#local-development)
- [Code Structure](#code-structure)
- [Contributing](#contributing)

## Development Environment

### Prerequisites

- **Go**: 1.21 or later
- **Make**: GNU Make 3.81+
- **Git**: For version control
- **Docker**: For integration tests
- **Firecracker**: For VM backend

### Tool Installation

```bash
# Install Go (Ubuntu/Debian)
sudo apt update
sudo apt install -y golang-go

# Install build dependencies
sudo apt install -y make gcc sqlite3

# Install Firecracker
ARCH="$(uname -m)"
latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} https://github.com/firecracker-microvm/firecracker/releases/latest))
curl -L https://github.com/firecracker-microvm/firecracker/releases/download/${latest}/firecracker-${latest}-${ARCH}.tgz | tar -xz
sudo mv release-${latest}-${ARCH}/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install gotest.tools/gotestsum@latest
```

## Building from Source

### Clone Repository

```bash
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald
```

### Build Commands

```bash
# Standard build
make build

# Build with version info
make build VERSION=0.2.0-dev

# Build with race detector (development only)
make build-race

# Cross-compilation
GOOS=linux GOARCH=amd64 make build
```

### Build Tags

Available build tags:
- `sqlite_json`: Enable SQLite JSON support
- `sqlite_fts5`: Enable full-text search
- `integration`: Include integration tests

Example:
```bash
go build -tags "sqlite_json integration" ./cmd/api
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary |
| `make test` | Run unit tests |
| `make test-integration` | Run integration tests |
| `make test-race` | Run tests with race detector |
| `make bench` | Run benchmarks |
| `make lint` | Run golangci-lint |
| `make clean` | Clean build artifacts |
| `make install` | Install binary and systemd service |
| `make proto` | Generate protobuf code |

## Testing

### Unit Tests

Run unit tests:
```bash
make test

# With coverage
make test-coverage

# Specific package
go test ./internal/service/...

# Verbose output
go test -v ./...
```

### Integration Tests

Integration tests require Docker and elevated privileges:
```bash
# Run all integration tests
sudo make test-integration

# Run specific integration test
sudo go test -tags integration ./tests/integration -run TestVMLifecycle
```

### Test Structure

Tests follow Go conventions:
- Unit tests: `*_test.go` alongside source files
- Integration tests: `tests/integration/`
- Mocks: Generated with `mockgen`
- Fixtures: `testdata/` directories

Example unit test:
```go
func TestVMService_CreateVm(t *testing.T) {
    // Setup
    backend := mocks.NewMockBackend(t)
    repo := mocks.NewMockVMRepository(t)
    service := NewVMService(backend, logger, nil, nil, repo)
    
    // Test
    req := &connect.Request[metaldv1.CreateVmRequest]{
        Msg: &metaldv1.CreateVmRequest{
            Config: &metaldv1.VmConfig{
                Cpu: &metaldv1.CpuConfig{VcpuCount: 2},
                Memory: &metaldv1.MemoryConfig{SizeBytes: 1<<30},
            },
        },
    }
    
    backend.EXPECT().CreateVM(mock.Any, mock.Any).Return("vm-123", nil)
    repo.EXPECT().CreateVM(mock.Any, mock.Any, mock.Any, mock.Any).Return(nil)
    
    resp, err := service.CreateVm(context.Background(), req)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "vm-123", resp.Msg.VmId)
}
```

### Benchmarks

Run performance benchmarks:
```bash
# All benchmarks
make bench

# Specific benchmark
go test -bench=BenchmarkVMCreation -benchmem ./internal/service

# With profiling
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof ./internal/service
```

## Local Development

### Running Locally

1. **Minimal Setup**:
```bash
# Create required directories
sudo mkdir -p /opt/metald/data /srv/jailer /opt/vm-assets

# Set permissions
sudo chown -R $USER:$USER /opt/metald /srv/jailer

# Run with mock dependencies
UNKEY_METALD_BILLING_MOCK_MODE=true \
UNKEY_METALD_ASSETMANAGER_ENABLED=false \
UNKEY_METALD_JAILER_UID=$(id -u) \
UNKEY_METALD_JAILER_GID=$(id -g) \
./metald
```

2. **With Dependencies**:
```bash
# Start billaged (mock mode)
cd ../billaged && ./billaged --mock &

# Start assetmanagerd
cd ../assetmanagerd && ./assetmanagerd &

# Run metald
UNKEY_METALD_BILLING_ENABLED=true \
UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081 \
UNKEY_METALD_ASSETMANAGER_ENABLED=true \
UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://localhost:8083 \
./metald
```

### Development Configuration

Create `.env.development`:
```bash
# Development settings
UNKEY_METALD_SERVER_PORT=8080
UNKEY_METALD_LOG_LEVEL=debug

# Use local paths
UNKEY_METALD_DATABASE_DATA_DIR=/tmp/metald/data
UNKEY_METALD_JAILER_CHROOT_BASE_DIR=/tmp/metald/jailer

# Mock external services
UNKEY_METALD_BILLING_MOCK_MODE=true
UNKEY_METALD_ASSETMANAGER_ENABLED=false

# Development UID/GID
UNKEY_METALD_JAILER_UID=1000
UNKEY_METALD_JAILER_GID=1000

# Enable debug endpoints
UNKEY_METALD_ENABLE_DEBUG=true
```

### Using the Example Client

The example client demonstrates API usage:

```bash
cd contrib/example-client

# List available actions
go run main.go -help

# Create and boot a VM
go run main.go -action create-and-boot

# Create with custom config
go run main.go -action create -vm-id my-test-vm

# List VMs
go run main.go -action list

# Get VM info
go run main.go -action info -vm-id my-test-vm
```

## Code Structure

### Directory Layout

```
metald/
├── cmd/
│   └── api/                 # Main application entry point
│       └── main.go         # Application initialization
├── contrib/
│   ├── example-client/     # Example API client
│   ├── grafana-dashboards/ # Monitoring dashboards
│   └── systemd/           # Service definitions
├── gen/                    # Generated protobuf code
├── internal/
│   ├── assetmanager/      # Asset management client
│   ├── backend/           # VM backend implementations
│   │   ├── types/        # Backend interface
│   │   ├── firecracker/  # Firecracker implementation
│   │   └── cloudhypervisor/ # Cloud Hypervisor (future)
│   ├── billing/           # Billing service integration
│   ├── config/            # Configuration management
│   ├── database/          # SQLite database layer
│   ├── health/            # Health check handlers
│   ├── jailer/            # Security isolation
│   ├── network/           # Network management
│   ├── observability/     # Metrics and tracing
│   └── service/           # API service handlers
├── proto/                  # Protobuf definitions
├── tests/                  # Integration tests
├── Makefile               # Build automation
└── go.mod                 # Go module definition
```

### Key Interfaces

#### Backend Interface
[backend.go:11-45](../../../metald/internal/backend/types/backend.go#L11-L45)
```go
type Backend interface {
    Initialize() error
    CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error)
    DeleteVM(ctx context.Context, vmID string) error
    BootVM(ctx context.Context, vmID string) error
    ShutdownVM(ctx context.Context, vmID string) error
    // ... other operations
}
```

#### Repository Interface
```go
type VMRepository interface {
    CreateVM(vmID, customerID string, config *metaldv1.VmConfig, state metaldv1.VmState) error
    GetVM(vmID string) (*VM, error)
    UpdateVMState(vmID string, state metaldv1.VmState, processID *string) error
    ListVMs(customerID *string, states []metaldv1.VmState, limit, offset int) ([]*VM, error)
    DeleteVM(vmID string) error
}
```

### Adding New Features

1. **New RPC Method**:
   - Add to `proto/vmprovisioner/v1/vm.proto`
   - Regenerate code: `make proto`
   - Implement handler in `internal/service/vm.go`
   - Add tests

2. **New Backend**:
   - Implement `Backend` interface
   - Add to backend factory in `main.go`
   - Add configuration support
   - Write integration tests

3. **New Metric**:
   - Define in `internal/observability/`
   - Initialize in service
   - Record at appropriate points
   - Add to documentation

### Code Style

Follow Go conventions:
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use structured logging with slog
- Add comments for exported functions
- Keep functions focused and small

### Error Handling

Use ConnectRPC error codes:
```go
if err != nil {
    if errors.Is(err, ErrNotFound) {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    return nil, connect.NewError(connect.CodeInternal, err)
}
```

## Contributing

### Development Workflow

1. **Fork and Clone**:
```bash
git clone https://github.com/YOUR_USERNAME/unkey
cd go/deploy/metald
git remote add upstream https://github.com/unkeyed/unkey
```

2. **Create Feature Branch**:
```bash
git checkout -b feature/my-feature
```

3. **Make Changes**:
   - Write code
   - Add tests
   - Update documentation

4. **Run Tests**:
```bash
make test
make lint
```

5. **Commit**:
```bash
git add .
git commit -m "feat: add new feature"
```

6. **Push and PR**:
```bash
git push origin feature/my-feature
# Create PR on GitHub
```

### Commit Message Format

Follow conventional commits:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `refactor:` Code refactoring
- `test:` Test changes
- `chore:` Maintenance

### Code Review Checklist

- [ ] Tests pass
- [ ] Code follows style guide
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Performance impact considered
- [ ] Error handling complete
- [ ] Logging appropriate

### Debugging Tips

1. **Enable Debug Logging**:
```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

2. **Use Delve Debugger**:
```bash
dlv debug ./cmd/api -- -config=dev.yaml
```

3. **Trace Requests**:
```bash
# Enable trace logging
export UNKEY_METALD_TRACE_REQUESTS=true
```

4. **Inspect Database**:
```bash
sqlite3 /opt/metald/data/vms.db
.tables
.schema vms
SELECT * FROM vms;
```

### Performance Profiling

1. **CPU Profiling**:
```go
import _ "net/http/pprof"

// In main()
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

2. **Memory Profiling**:
```bash
go test -memprofile mem.prof -bench .
go tool pprof mem.prof
```

3. **Trace Analysis**:
```bash
go test -trace trace.out
go tool trace trace.out
```