# Development Setup

This document provides comprehensive guidance for setting up a development environment, building, testing, and contributing to metald.

## Development Environment

### Prerequisites

#### System Requirements
- **Go**: 1.24.4+ ([Installation Guide](https://golang.org/doc/install))
- **Linux**: Ubuntu 20.04+, CentOS 8+, or similar (required for VM operations)
- **Docker**: For integration testing and builderd compatibility
- **Git**: For version control and contribution workflow

#### Required Tools

```bash
# Install Go tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Install buf for protobuf management
curl -fsSL https://buf.build/install.sh | sh
```

#### System Dependencies

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y \
    build-essential \
    libsqlite3-dev \
    iproute2 \
    iptables \
    bridge-utils \
    curl \
    jq

# CentOS/RHEL
sudo yum groupinstall -y "Development Tools"
sudo yum install -y \
    sqlite-devel \
    iproute \
    iptables \
    bridge-utils \
    curl \
    jq
```

### Project Setup

#### Clone Repository

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald

# Initialize Go modules
go mod download
go mod verify
```

#### Install Firecracker (Development)

```bash
# Download and install Firecracker for development
curl -fsSL https://github.com/firecracker-microvm/firecracker/releases/download/v1.4.1/firecracker-v1.4.1-x86_64.tgz | tar -xz
sudo mv firecracker-v1.4.1-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker

# Verify installation
firecracker --version
```

#### Development Network Setup

```bash
# Create development bridge (one-time setup)
sudo ip link add name br-dev type bridge
sudo ip addr add 172.32.0.1/24 dev br-dev
sudo ip link set br-dev up

# Enable forwarding for development
sudo sysctl -w net.ipv4.ip_forward=1

# Add NAT rule for development VMs
sudo iptables -t nat -A POSTROUTING -s 172.32.0.0/24 -o eth0 -j MASQUERADE
```

### IDE Configuration

#### VS Code Setup

**File**: `.vscode/settings.json`
```json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.formatTool": "goimports",
    "go.generateTestsFlags": [
        "-template_dir", "testdata/templates"
    ],
    "files.exclude": {
        "**/build": true,
        "**/gen": true
    }
}
```

**File**: `.vscode/launch.json`
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Metald",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/metald",
            "env": {
                "UNKEY_METALD_PORT": "8080",
                "UNKEY_METALD_ADDRESS": "127.0.0.1",
                "UNKEY_METALD_BACKEND": "firecracker",
                "UNKEY_METALD_TLS_MODE": "disabled",
                "UNKEY_METALD_BILLING_MOCK_MODE": "true",
                "UNKEY_METALD_ASSETMANAGER_ENABLED": "false",
                "UNKEY_METALD_DATA_DIR": "/tmp/metald-dev",
                "UNKEY_METALD_NETWORK_BRIDGE_IPV4": "172.32.0.1/24",
                "UNKEY_METALD_NETWORK_VM_SUBNET_IPV4": "172.32.0.0/24"
            },
            "console": "integratedTerminal",
            "asRoot": true
        }
    ]
}
```

#### GoLand/IntelliJ Setup

**File**: `.idea/runConfigurations/Metald_Development.xml`
```xml
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Metald Development" type="GoApplicationRunConfiguration">
    <module name="metald" />
    <working_directory value="$PROJECT_DIR$" />
    <kind value="PACKAGE" />
    <package value="github.com/unkeyed/unkey/go/deploy/metald/cmd/metald" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$" />
    <envs>
      <env name="UNKEY_METALD_PORT" value="8080" />
      <env name="UNKEY_METALD_TLS_MODE" value="disabled" />
      <env name="UNKEY_METALD_BILLING_MOCK_MODE" value="true" />
      <env name="UNKEY_METALD_DATA_DIR" value="/tmp/metald-dev" />
    </envs>
    <method v="2" />
  </configuration>
</component>
```

## Build Instructions

### Build System

The project uses a Makefile for build automation:

#### Build Targets

```bash
# Build main binary
make build

# Build with debug symbols
make build-debug

# Build all components (including client)
make build-all

# Clean build artifacts
make clean

# Install to system (requires root)
sudo make install

# Generate protobuf code
make generate

# Format code
make fmt

# Run linter
make lint

# Run tests
make test
```

#### Manual Build Commands

```bash
# Build metald binary
go build -o build/metald ./cmd/metald

# Build with version info
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.version=$VERSION" -o build/metald ./cmd/metald

# Build client library
cd client
go build -o build/metald-cli ./cmd/metald-cli

# Build with race detection (development)
go build -race -o build/metald ./cmd/metald
```

### Cross-Compilation

```bash
# Build for different architectures
GOOS=linux GOARCH=amd64 go build -o build/metald-linux-amd64 ./cmd/metald
GOOS=linux GOARCH=arm64 go build -o build/metald-linux-arm64 ./cmd/metald

# Build statically linked binary
CGO_ENABLED=0 go build -ldflags "-extldflags '-static'" -o build/metald-static ./cmd/metald
```

### Development Build Configuration

**File**: `scripts/dev-build.sh`
```bash
#!/bin/bash
set -e

# Development build script
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse HEAD)

# Build flags
LDFLAGS="-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT"

# Build with development flags
go build -race -ldflags "$LDFLAGS" -o build/metald ./cmd/metald

echo "Built metald $VERSION at $BUILD_TIME"
```

## Testing

### Test Structure

The project includes multiple test types:

```
metald/
├── internal/
│   ├── service/
│   │   ├── vm_test.go              # Unit tests
│   │   ├── vm_cleanup_test.go      # Cleanup logic tests
│   │   └── vm_integration_test.go  # Integration tests
│   ├── backend/
│   │   └── firecracker/
│   │       ├── sdk_client_v4_test.go
│   │       └── automatic_build_test.go
│   ├── network/
│   │   └── idgen_test.go
│   └── config/
│       └── config_test.go
├── client/
│   └── example_test.go
└── testdata/
    ├── configs/                    # Test VM configurations
    ├── assets/                     # Test assets
    └── scripts/                    # Test scripts
```

### Running Tests

#### Unit Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific test package
go test ./internal/service/

# Run specific test
go test -run TestVMService_CreateVm ./internal/service/

# Verbose test output
go test -v ./internal/service/

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Integration Tests

```bash
# Run integration tests (requires root for networking)
sudo go test -tags=integration ./internal/service/

# Run with custom environment
sudo UNKEY_METALD_TEST_BRIDGE=br-test go test -tags=integration ./internal/service/

# Run performance tests
go test -bench=. ./internal/service/
```

#### Mock Testing

The project uses mocks for external dependencies:

**AssetManager Mock**: [assetmanager/client.go:337](../../internal/assetmanager/client.go#L337)
```go
// noopClient is used when assetmanagerd integration is disabled
type noopClient struct{}
```

**Billing Mock**: [billing/client.go:38](../../internal/billing/client.go#L38)
```go
// MockBillingClient provides a mock implementation for development and testing
type MockBillingClient struct {
    logger *slog.Logger
}
```

#### Test Configuration

**File**: `testdata/configs/test-vm.json`
```json
{
  "cpu": {
    "vcpu_count": 1
  },
  "memory": {
    "size_bytes": 268435456
  },
  "boot": {
    "kernel_path": "/opt/test/vmlinux.bin",
    "kernel_args": "console=ttyS0 quiet"
  },
  "storage": [{
    "id": "rootfs",
    "path": "/opt/test/rootfs.ext4",
    "is_root_device": true,
    "read_only": false
  }],
  "console": {
    "enabled": true,
    "console_type": "serial"
  }
}
```

### Benchmarking

#### Performance Tests

```bash
# Run benchmarks
go test -bench=. ./internal/service/

# Run specific benchmark
go test -bench=BenchmarkVMCreation ./internal/service/

# Benchmark with memory profiling
go test -bench=. -memprofile=mem.prof ./internal/service/

# Benchmark with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/service/

# Analyze profiles
go tool pprof mem.prof
go tool pprof cpu.prof
```

#### Load Testing

**File**: `scripts/load-test.sh`
```bash
#!/bin/bash
# Load test VM creation/deletion

ENDPOINT="http://localhost:8080"
TOKEN="test-token"
CONCURRENT=10
ITERATIONS=100

# Function to create and delete VM
test_vm_lifecycle() {
    local vm_id="test-vm-$(uuidgen)"
    
    # Create VM
    curl -s -X POST "$ENDPOINT/vmprovisioner.v1.VmService/CreateVm" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"vm_id\":\"$vm_id\",\"config\":{\"cpu\":{\"vcpu_count\":1},\"memory\":{\"size_bytes\":268435456},\"boot\":{\"kernel_path\":\"/opt/test/vmlinux.bin\"},\"storage\":[{\"path\":\"/opt/test/rootfs.ext4\",\"is_root_device\":true}]}}"
    
    # Delete VM
    curl -s -X POST "$ENDPOINT/vmprovisioner.v1.VmService/DeleteVm" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"vm_id\":\"$vm_id\",\"force\":true}"
}

# Run concurrent tests
for i in $(seq 1 $CONCURRENT); do
    for j in $(seq 1 $ITERATIONS); do
        test_vm_lifecycle &
    done
done

wait
echo "Load test completed"
```

## Local Development

### Development Environment

#### Environment Variables

**File**: `scripts/dev-env.sh`
```bash
#!/bin/bash
# Development environment setup

export UNKEY_METALD_PORT=8080
export UNKEY_METALD_ADDRESS=127.0.0.1
export UNKEY_METALD_BACKEND=firecracker
export UNKEY_METALD_TLS_MODE=disabled
export UNKEY_METALD_DATA_DIR=/tmp/metald-dev
export UNKEY_METALD_OTEL_ENABLED=false

# Mock external services
export UNKEY_METALD_BILLING_ENABLED=true
export UNKEY_METALD_BILLING_MOCK_MODE=true
export UNKEY_METALD_ASSETMANAGER_ENABLED=false

# Development network
export UNKEY_METALD_NETWORK_ENABLED=true
export UNKEY_METALD_NETWORK_BRIDGE_IPV4=172.32.0.1/24
export UNKEY_METALD_NETWORK_VM_SUBNET_IPV4=172.32.0.0/24
export UNKEY_METALD_NETWORK_BRIDGE=br-dev

# Jailer configuration for development
export UNKEY_METALD_JAILER_UID=1000
export UNKEY_METALD_JAILER_GID=1000
export UNKEY_METALD_JAILER_CHROOT_DIR=/tmp/jailer-dev

echo "Development environment configured"
```

#### Running Local Development Server

```bash
# Source development environment
source scripts/dev-env.sh

# Create development directories
mkdir -p /tmp/metald-dev /tmp/jailer-dev

# Build and run
make build && sudo ./build/metald

# Or run with go run (for debugging)
sudo go run ./cmd/metald
```

### Client Development

#### Using the Go Client Library

**File**: `examples/basic_client.go`
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/unkeyed/unkey/go/deploy/metald/client"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create client
    c, err := client.New(ctx, client.Config{
        ServerAddress: "http://localhost:8080",
        CustomerToken: "test-token",
        TLSMode:      "disabled",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Create VM
    vm, err := c.CreateVM(ctx, &client.CreateVMRequest{
        Config: &client.VMConfig{
            CPU: &client.CPUConfig{
                VCPUCount: 1,
            },
            Memory: &client.MemoryConfig{
                SizeBytes: 256 * 1024 * 1024, // 256MB
            },
            Boot: &client.BootConfig{
                KernelPath: "/opt/test/vmlinux.bin",
            },
            Storage: []*client.StorageDevice{{
                Path:         "/opt/test/rootfs.ext4",
                IsRootDevice: true,
            }},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Created VM: %s\n", vm.VmId)

    // Boot VM
    _, err = c.BootVM(ctx, &client.BootVMRequest{
        VmId: vm.VmId,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Booted VM: %s\n", vm.VmId)

    // Get VM info
    info, err := c.GetVMInfo(ctx, &client.GetVMInfoRequest{
        VmId: vm.VmId,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("VM State: %s\n", info.State)
    fmt.Printf("VM IP: %s\n", info.NetworkInfo.IpAddress)

    // Cleanup
    _, err = c.DeleteVM(ctx, &client.DeleteVMRequest{
        VmId: vm.VmId,
        Force: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Deleted VM: %s\n", vm.VmId)
}
```

#### CLI Development

```bash
# Build CLI tool
cd client/cmd/metald-cli
go build -o metald-cli

# Test CLI commands
./metald-cli --server http://localhost:8080 --token test-token list-vms
./metald-cli --server http://localhost:8080 --token test-token create-vm --config ../../examples/configs/minimal.json
```

### Service Integration Testing

#### Mock Service Setup

**File**: `scripts/start-mock-services.sh`
```bash
#!/bin/bash
# Start mock services for integration testing

# Start mock assetmanagerd
cd ../../assetmanagerd
UNKEY_ASSETMANAGERD_PORT=8083 UNKEY_ASSETMANAGERD_TLS_MODE=disabled go run ./cmd/assetmanagerd &
ASSET_PID=$!

# Start mock billaged
cd ../billaged
UNKEY_BILLAGED_PORT=8081 UNKEY_BILLAGED_TLS_MODE=disabled go run ./cmd/billaged &
BILLING_PID=$!

# Wait for services to start
sleep 5

echo "Mock services started:"
echo "AssetManager: http://localhost:8083 (PID: $ASSET_PID)"
echo "Billing: http://localhost:8081 (PID: $BILLING_PID)"

# Cleanup function
cleanup() {
    echo "Stopping mock services..."
    kill $ASSET_PID $BILLING_PID 2>/dev/null
    exit 0
}

trap cleanup SIGINT SIGTERM

# Keep script running
wait
```

### Debugging

#### Debug Configuration

**VS Code Debug Settings**:
```json
{
    "name": "Debug Metald with Delve",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/cmd/metald",
    "env": {
        "UNKEY_METALD_PORT": "8080",
        "UNKEY_METALD_TLS_MODE": "disabled",
        "UNKEY_METALD_DATA_DIR": "/tmp/metald-debug"
    },
    "args": [],
    "showLog": true,
    "logOutput": "rpc",
    "asRoot": true
}
```

#### Debug Logging

```bash
# Enable debug logging
export UNKEY_METALD_LOG_LEVEL=debug

# Enable trace logging for specific components
export UNKEY_METALD_TRACE_BACKEND=true
export UNKEY_METALD_TRACE_NETWORK=true

# Run with debug output
sudo go run ./cmd/metald 2>&1 | jq '.'
```

#### Profiling

```bash
# Enable pprof endpoints
export UNKEY_METALD_PPROF_ENABLED=true

# Build with profiling
go build -tags pprof -o build/metald ./cmd/metald

# Collect profiles
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

## Code Style and Standards

### Go Conventions

#### Code Formatting

```bash
# Format code with goimports
goimports -w .

# Format with gofmt
gofmt -w .

# Use consistent import grouping
# 1. Standard library
# 2. Third-party packages
# 3. Local packages
```

#### Linting Rules

**File**: `.golangci.yml`
```yaml
run:
  timeout: 5m
  tests: true

linters-settings:
  govet:
    check-shadowing: true
  gocyclo:
    min-complexity: 15
  maligned:
    suggest-new: true
  dupl:
    threshold: 100
  goconst:
    min-len: 2
    min-occurrences: 2

linters:
  enable:
    - goimports
    - govet
    - errcheck
    - deadcode
    - varcheck
    - ineffassign
    - typecheck
    - golint
    - gocyclo
    - gofmt
    - misspell
    - unparam
    - unused
    - gosimple
    - staticcheck

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
        - dupl
        - gosec
```

#### Comment Standards

```go
// Package service implements the VM management business logic layer.
// It provides unified VM lifecycle operations across different hypervisor backends
// with multi-tenant security and comprehensive resource management.
package service

// VMService implements the VmServiceHandler interface.
// It provides VM lifecycle management with backend abstraction,
// billing integration, and database state persistence.
type VMService struct {
    backend          types.Backend
    logger           *slog.Logger
    metricsCollector *billing.MetricsCollector
    vmMetrics        *observability.VMMetrics
    vmRepo           *database.VMRepository
    tracer           trace.Tracer
}

// CreateVm creates a new VM instance with the specified configuration.
// It validates the configuration, prepares required assets via assetmanagerd,
// creates the VM using the configured backend, and persists state to the database.
//
// The operation includes:
//   - Customer authentication and authorization validation
//   - VM configuration validation and normalization
//   - Asset preparation and lease acquisition
//   - Backend VM creation with error handling and cleanup
//   - Database state persistence with transaction safety
//
// Returns the created VM ID and initial state, or an error if creation fails.
func (s *VMService) CreateVm(ctx context.Context, req *connect.Request[metaldv1.CreateVmRequest]) (*connect.Response[metaldv1.CreateVmResponse], error) {
    // Implementation follows...
}
```

### Project Structure

#### Directory Organization

```
metald/
├── cmd/                           # Main applications
│   ├── metald/                    # Main service binary
│   └── metald-init/               # Init process for containers
├── internal/                      # Private application code
│   ├── service/                   # Business logic layer
│   ├── backend/                   # Hypervisor backends
│   │   ├── firecracker/           # Firecracker implementation
│   │   ├── cloudhypervisor/       # Cloud Hypervisor (planned)
│   │   └── types/                 # Backend interface
│   ├── assetmanager/              # AssetManager client
│   ├── billing/                   # Billing client
│   ├── config/                    # Configuration management
│   ├── database/                  # Database layer
│   ├── network/                   # Network management
│   ├── observability/             # Metrics and tracing
│   ├── reconciler/                # State reconciliation
│   └── jailer/                    # Security isolation
├── client/                        # Client library
│   ├── cmd/metald-cli/            # CLI tool
│   └── examples/                  # Usage examples
├── proto/                         # Protocol buffer definitions
├── gen/                           # Generated code
├── docs/                          # Documentation
├── testdata/                      # Test fixtures
├── scripts/                       # Build and deployment scripts
└── contrib/                       # Distribution-specific files
    ├── systemd/                   # Systemd service files
    └── grafana-dashboards/        # Monitoring dashboards
```

#### Naming Conventions

- **Packages**: lowercase, single word when possible
- **Files**: snake_case with descriptive names
- **Types**: PascalCase with clear purpose
- **Functions**: PascalCase for exported, camelCase for internal
- **Constants**: SCREAMING_SNAKE_CASE for package-level, PascalCase for exported
- **Variables**: camelCase with meaningful names

### Error Handling

#### Error Types

```go
// Define custom error types for different failure modes
type VMError struct {
    Op       string    // Operation that failed
    VMId     string    // VM identifier
    Err      error     // Underlying error
}

func (e *VMError) Error() string {
    return fmt.Sprintf("vm %s %s: %v", e.VMId, e.Op, e.Err)
}

// Use errors.Is and errors.As for error checking
func isVMNotFound(err error) bool {
    var vmErr *VMError
    return errors.As(err, &vmErr) && errors.Is(vmErr.Err, ErrVMNotFound)
}
```

#### Error Context

```go
// Add context to errors with consistent formatting
func (s *VMService) CreateVm(ctx context.Context, req *connect.Request[metaldv1.CreateVmRequest]) (*connect.Response[metaldv1.CreateVmResponse], error) {
    vmID, err := s.backend.CreateVM(ctx, config)
    if err != nil {
        // Add operation context
        return nil, fmt.Errorf("failed to create vm %s: %w", vmID, err)
    }
    
    if err := s.vmRepo.CreateVMWithContext(ctx, vmID, customerID, config, state); err != nil {
        // Clean up on database failure
        if cleanupErr := s.backend.DeleteVM(ctx, vmID); cleanupErr != nil {
            // Log cleanup failure but return original error
            s.logger.Error("failed to cleanup vm after database error",
                "vm_id", vmID,
                "cleanup_error", cleanupErr,
                "original_error", err)
        }
        return nil, fmt.Errorf("failed to persist vm %s: %w", vmID, err)
    }
    
    return &metaldv1.CreateVmResponse{VmId: vmID, State: state}, nil
}
```

## Contributing

### Development Workflow

#### Git Workflow

```bash
# Create feature branch
git checkout -b feature/vm-pause-resume

# Make changes with atomic commits
git add internal/service/vm.go
git commit -m "service: add VM pause/resume functionality

- Implement PauseVm and ResumeVm RPC methods
- Add backend pause/resume interface methods
- Include customer ownership validation
- Add metrics collection for pause/resume operations

Fixes #123"

# Push branch and create PR
git push origin feature/vm-pause-resume
```

#### Commit Message Format

```
<type>(<scope>): <description>

<body>

<footer>
```

**Types**: feat, fix, docs, style, refactor, test, chore
**Scopes**: service, backend, client, config, network, billing, docs

#### Pre-commit Hooks

**File**: `.git/hooks/pre-commit`
```bash
#!/bin/bash
# Pre-commit hooks for code quality

set -e

# Run tests
echo "Running tests..."
make test

# Run linter
echo "Running linter..."
make lint

# Check formatting
echo "Checking formatting..."
if ! make fmt-check; then
    echo "Code formatting issues found. Run 'make fmt' to fix."
    exit 1
fi

# Generate code if needed
if git diff --cached --name-only | grep -q "\.proto$"; then
    echo "Proto files changed, regenerating code..."
    make generate
    git add gen/
fi

echo "Pre-commit checks passed"
```

### Testing Guidelines

#### Test Structure

```go
func TestVMService_CreateVm(t *testing.T) {
    tests := []struct {
        name           string
        request        *metaldv1.CreateVmRequest
        setupMocks     func(*testing.T) (*mockBackend, *mockBillingClient)
        expectedError  string
        expectedVMId   string
        expectedState  metaldv1.VmState
    }{
        {
            name: "successful vm creation",
            request: &metaldv1.CreateVmRequest{
                Config: validVMConfig(),
            },
            setupMocks: func(t *testing.T) (*mockBackend, *mockBillingClient) {
                backend := &mockBackend{}
                backend.On("CreateVM", mock.Anything, mock.Anything).Return("vm-123", nil)
                
                billing := &mockBillingClient{}
                billing.On("NotifyVmStarted", mock.Anything, mock.Anything, mock.Anything).Return(nil)
                
                return backend, billing
            },
            expectedVMId:  "vm-123",
            expectedState: metaldv1.VmState_VM_STATE_CREATED,
        },
        {
            name: "invalid vm config",
            request: &metaldv1.CreateVmRequest{
                Config: invalidVMConfig(),
            },
            expectedError: "vm config is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### Integration Test Patterns

```go
//go:build integration

func TestVMLifecycle_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Require root for network operations
    if os.Geteuid() != 0 {
        t.Skip("integration tests require root privileges")
    }
    
    // Setup test environment
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // Create test service with real backends
    service := setupIntegrationService(t)
    defer cleanupIntegrationService(t, service)
    
    // Test full VM lifecycle
    vmID := createTestVM(t, service)
    defer deleteTestVM(t, service, vmID)
    
    bootTestVM(t, service, vmID)
    validateVMRunning(t, service, vmID)
    shutdownTestVM(t, service, vmID)
}
```

### Documentation Standards

#### Code Documentation

```go
// Package database provides VM state persistence and repository operations.
// It implements SQLite-based storage with customer isolation, transaction safety,
// and automatic state reconciliation for the metald service.
//
// The database layer includes:
//   - VM state persistence with protobuf serialization
//   - Customer-scoped queries for multi-tenant isolation  
//   - Soft delete support for audit trails
//   - Database migration and schema management
//
// All database operations include proper error handling, transaction management,
// and logging for production observability.
package database

// VMRepository handles VM state persistence operations with customer isolation.
// It provides CRUD operations for VM records with automatic serialization
// of protobuf configurations and multi-tenant security boundaries.
//
// The repository includes:
//   - Customer-scoped VM queries and updates
//   - Transactional state updates with rollback support
//   - Soft delete operations for audit compliance
//   - Background reconciliation support
//
// All methods include context support for cancellation and tracing.
type VMRepository struct {
    db     *Database
    logger *slog.Logger
}
```

#### API Documentation

All API changes require corresponding documentation updates:

1. Update proto comments
2. Update API documentation
3. Update client examples
4. Update integration tests

### Performance Guidelines

#### Memory Management

```go
// Use object pools for frequently allocated objects
var vmConfigPool = sync.Pool{
    New: func() interface{} {
        return &metaldv1.VmConfig{}
    },
}

func (s *VMService) processVMConfig(config *metaldv1.VmConfig) error {
    // Get from pool
    working := vmConfigPool.Get().(*metaldv1.VmConfig)
    defer vmConfigPool.Put(working)
    
    // Reset and use
    working.Reset()
    proto.Merge(working, config)
    
    // Process working config
    return s.validateConfig(working)
}
```

#### Concurrency Patterns

```go
// Use worker pools for concurrent operations
type VMWorkerPool struct {
    workers   int
    taskChan  chan VMTask
    resultChan chan VMResult
    wg        sync.WaitGroup
}

func (p *VMWorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
}

func (p *VMWorkerPool) worker() {
    defer p.wg.Done()
    for task := range p.taskChan {
        result := task.Execute()
        p.resultChan <- result
    }
}
```

### Security Guidelines

#### Input Validation

```go
func (s *VMService) validateVMConfig(config *metaldv1.VmConfig) error {
    // Validate required fields
    if config.GetCpu() == nil {
        return fmt.Errorf("cpu configuration is required")
    }
    
    // Validate value ranges
    if cpu := config.GetCpu(); cpu.GetVcpuCount() <= 0 || cpu.GetVcpuCount() > 128 {
        return fmt.Errorf("vcpu_count must be between 1 and 128, got %d", cpu.GetVcpuCount())
    }
    
    // Validate memory limits (prevent resource exhaustion)
    if mem := config.GetMemory(); mem.GetSizeBytes() > 64*1024*1024*1024 { // 64GB max
        return fmt.Errorf("memory size exceeds maximum allowed (64GB)")
    }
    
    // Validate file paths (prevent directory traversal)
    for _, storage := range config.GetStorage() {
        if strings.Contains(storage.GetPath(), "..") {
            return fmt.Errorf("storage path contains invalid characters: %s", storage.GetPath())
        }
    }
    
    return nil
}
```

#### Authentication Context

```go
func (s *VMService) validateVMOwnership(ctx context.Context, vmID string) error {
    // Extract customer from authenticated context
    customerID, err := ExtractCustomerID(ctx)
    if err != nil {
        return connect.NewError(connect.CodeUnauthenticated, err)
    }
    
    // Verify VM ownership
    vm, err := s.vmRepo.GetVMWithContext(ctx, vmID)
    if err != nil {
        return connect.NewError(connect.CodeNotFound, fmt.Errorf("vm not found"))
    }
    
    if vm.CustomerID != customerID {
        s.logger.Warn("SECURITY: attempted access to VM owned by different customer",
            "vm_id", vmID,
            "owner", vm.CustomerID,
            "accessor", customerID)
        return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("vm not found"))
    }
    
    return nil
}
```

This comprehensive development guide provides everything needed to contribute effectively to metald, from environment setup through testing, documentation, and security best practices.