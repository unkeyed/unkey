# Builderd Development Setup

## Getting Started

### Prerequisites

- **Go**: Version 1.21+ installed
- **Docker**: Version 20.10+ for build execution
- **Make**: For build automation
- **SPIRE** (optional): For SPIFFE/mTLS testing
- **Git**: For version control

### Repository Setup

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey.git
cd unkey/go/deploy/builderd

# Install dependencies
go mod download

# Verify setup
go mod verify
```

## Build Instructions

### Building the Binary

```bash
# Build the binary
make build

# Output location
ls -la build/builderd
```

The binary is built with version information:
- Git commit hash
- Build timestamp
- Go version

### Installing with Systemd

```bash
# Build and install with systemd unit
make install

# This will:
# 1. Build the binary
# 2. Copy to /usr/local/bin/
# 3. Install systemd unit file
# 4. Create necessary directories
```

### Cross-Platform Builds

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o build/builderd-linux-amd64 ./cmd/builderd

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o build/builderd-linux-arm64 ./cmd/builderd

# macOS (for development only)
GOOS=darwin GOARCH=amd64 go build -o build/builderd-darwin-amd64 ./cmd/builderd
```

## Local Development Environment

### Basic Setup

1. **Create development directories**:
   ```bash
   mkdir -p /tmp/builderd/{workspace,rootfs,cache}
   ```

2. **Set development environment**:
   ```bash
   export UNKEY_BUILDERD_TLS_MODE=disabled
   export UNKEY_BUILDERD_STORAGE_BACKEND=local
   export UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/tmp/builderd/rootfs
   export UNKEY_BUILDERD_WORKSPACE_DIR=/tmp/builderd/workspace
   export UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd/cache
   export UNKEY_BUILDERD_ASSETMANAGER_ENABLED=false
   export UNKEY_BUILDERD_OTEL_ENABLED=false
   ```

3. **Run builderd**:
   ```bash
   go run cmd/builderd/main.go
   ```

### Development with Dependencies

#### Running with Docker

Ensure Docker daemon is running:
```bash
# Check Docker status
docker info

# If using Docker Desktop, ensure it's running
# For Linux, ensure dockerd is running
sudo systemctl status docker
```

#### Running with AssetManagerd

1. **Start assetmanagerd** (in separate terminal):
   ```bash
   cd ../assetmanagerd
   go run cmd/assetmanagerd/main.go
   ```

2. **Configure builderd**:
   ```bash
   export UNKEY_BUILDERD_ASSETMANAGER_ENABLED=true
   export UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=http://localhost:8083
   ```

#### Running with SPIFFE/SPIRE

1. **Start SPIRE agent**:
   ```bash
   # Using Docker
   docker run -d \
     --name spire-agent \
     -v /run/spire/sockets:/run/spire/sockets \
     spiffe/spire-agent:latest
   ```

2. **Configure builderd**:
   ```bash
   export UNKEY_BUILDERD_TLS_MODE=spiffe
   export UNKEY_BUILDERD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock
   ```

### Development Scripts

Create a `dev.sh` script for easy development:

```bash
#!/bin/bash
# dev.sh - Development helper script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Default action
ACTION=${1:-run}

case $ACTION in
  build)
    echo -e "${GREEN}Building builderd...${NC}"
    go build -o build/builderd ./cmd/builderd
    ;;
    
  run)
    echo -e "${GREEN}Running builderd in development mode...${NC}"
    export UNKEY_BUILDERD_TLS_MODE=disabled
    export UNKEY_BUILDERD_STORAGE_BACKEND=local
    export UNKEY_BUILDERD_ASSETMANAGER_ENABLED=false
    go run cmd/builderd/main.go
    ;;
    
  test)
    echo -e "${GREEN}Running tests...${NC}"
    go test ./...
    ;;
    
  clean)
    echo -e "${GREEN}Cleaning build artifacts...${NC}"
    rm -rf build/
    rm -rf /tmp/builderd/
    ;;
    
  *)
    echo "Usage: $0 {build|run|test|clean}"
    exit 1
    ;;
esac
```

## Testing Strategies

### Unit Testing

#### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Writing Tests

Example test for build service:

```go
// internal/service/builder_test.go
package service

import (
    "context"
    "testing"
    
    "connectrpc.com/connect"
    builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestBuilderService_CreateBuild(t *testing.T) {
    // Setup
    logger := slog.Default()
    config := &config.Config{
        Builder: config.BuilderConfig{
            MaxConcurrentBuilds: 5,
        },
    }
    service := NewBuilderService(logger, nil, config, nil)
    
    // Test case
    req := &builderv1.CreateBuildRequest{
        Config: &builderv1.BuildConfig{
            Tenant: &builderv1.TenantContext{
                TenantId: "test-tenant",
            },
            Source: &builderv1.BuildSource{
                SourceType: &builderv1.BuildSource_DockerImage{
                    DockerImage: &builderv1.DockerImageSource{
                        ImageUri: "alpine:latest",
                    },
                },
            },
        },
    }
    
    // Execute
    resp, err := service.CreateBuild(context.Background(), 
        connect.NewRequest(req))
    
    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Msg.BuildId)
    assert.Equal(t, builderv1.BuildState_BUILD_STATE_PENDING, 
        resp.Msg.State)
}
```

### Integration Testing

#### Docker Executor Test

```go
// internal/executor/docker_test.go
func TestDockerExecutor_ExtractImage(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Requires Docker daemon
    executor := NewDockerExecutor(logger, config, nil)
    
    req := &builderv1.CreateBuildRequest{
        Config: &builderv1.BuildConfig{
            Source: &builderv1.BuildSource{
                SourceType: &builderv1.BuildSource_DockerImage{
                    DockerImage: &builderv1.DockerImageSource{
                        ImageUri: "alpine:3.18",
                    },
                },
            },
        },
    }
    
    result, err := executor.ExtractDockerImage(
        context.Background(), req)
    
    require.NoError(t, err)
    assert.FileExists(t, result.RootfsPath)
}
```

#### End-to-End Test

```bash
# test/e2e/build_test.sh
#!/bin/bash

# Start builderd
./build/builderd &
BUILDERD_PID=$!

# Wait for startup
sleep 5

# Test build creation
grpcurl -plaintext \
  -d '{
    "config": {
      "tenant": {"tenant_id": "test"},
      "source": {
        "docker_image": {
          "image_uri": "alpine:latest"
        }
      }
    }
  }' \
  localhost:8082 \
  builder.v1.BuilderService/CreateBuild

# Cleanup
kill $BUILDERD_PID
```

### Testing with Mock Services

#### Mock AssetManagerd

```go
// test/mocks/assetmanager.go
type MockAssetManagerClient struct {
    RegisterFunc func(ctx context.Context, req *assetv1.RegisterAssetRequest) (*assetv1.RegisterAssetResponse, error)
}

func (m *MockAssetManagerClient) RegisterAsset(ctx context.Context, req *connect.Request[assetv1.RegisterAssetRequest]) (*connect.Response[assetv1.RegisterAssetResponse], error) {
    if m.RegisterFunc != nil {
        resp, err := m.RegisterFunc(ctx, req.Msg)
        if err != nil {
            return nil, err
        }
        return connect.NewResponse(resp), nil
    }
    return connect.NewResponse(&assetv1.RegisterAssetResponse{
        Asset: &assetv1.Asset{
            Id: "asset-123",
        },
    }), nil
}
```

## Debugging Techniques

### Debug Logging

Enable verbose logging:
```bash
export UNKEY_BUILDERD_LOG_LEVEL=debug
go run cmd/builderd/main.go
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/builderd/main.go

# Set breakpoints
(dlv) break main.main
(dlv) break service.(*BuilderService).CreateBuild

# Run
(dlv) continue
```

### Remote Debugging

For debugging in containers:

```dockerfile
FROM golang:1.21
RUN go install github.com/go-delve/delve/cmd/dlv@latest
WORKDIR /app
COPY . .
EXPOSE 8082 2345
CMD ["dlv", "debug", "./cmd/builderd", "--headless", "--listen=:2345", "--api-version=2"]
```

### Performance Profiling

Enable profiling endpoints:

```go
// Add to main.go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Profile the application:
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Development Tools

### Code Generation

```bash
# Generate protobuf code
make proto

# This runs:
# buf generate proto/
```

### Linting

```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Run linting
golangci-lint run ./...

# Fix issues automatically
golangci-lint run --fix ./...
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Use goimports for import organization
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

## Contributing Guidelines

### Code Style

1. **Follow Go conventions**:
   - Use `gofmt` for formatting
   - Follow effective Go guidelines
   - Use meaningful variable names

2. **Error Handling**:
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to create build: %w", err)
   }
   
   // Bad
   if err != nil {
       return err
   }
   ```

3. **Logging**:
   ```go
   // Use structured logging
   logger.InfoContext(ctx, "build created",
       slog.String("build_id", buildID),
       slog.String("tenant_id", tenantID),
       slog.Duration("duration", duration),
   )
   ```

4. **Comments**:
   - Add godoc comments for exported functions
   - Use AIDEV-NOTE for important implementation details
   - Explain complex logic inline

### Git Workflow

1. **Branch Naming**:
   - Feature: `feature/add-git-executor`
   - Bugfix: `fix/docker-pull-timeout`
   - Refactor: `refactor/tenant-isolation`

2. **Commit Messages**:
   - Use conventional commits
   - Examples:
     - `feat: add Git repository source support`
     - `fix: handle Docker auth for private registries`
     - `refactor: extract build metrics to separate package`

3. **Pull Request Process**:
   - Write clear PR description
   - Include test coverage
   - Update documentation
   - Request reviews

### Testing Requirements

1. **Unit Tests**: Required for all new code
2. **Integration Tests**: For external dependencies
3. **Coverage**: Maintain >80% coverage
4. **Performance**: No significant regression

## Common Development Tasks

### Adding a New Source Type

1. **Define protobuf message**:
   ```protobuf
   // proto/builder/v1/builder.proto
   message GitRepositorySource {
     string repository_url = 1;
     string ref = 2;
     GitAuth auth = 3;
   }
   ```

2. **Generate code**:
   ```bash
   make proto
   ```

3. **Implement executor**:
   ```go
   // internal/executor/git.go
   type GitExecutor struct {
       logger *slog.Logger
       config *config.Config
   }
   
   func (g *GitExecutor) Execute(ctx context.Context, 
       req *builderv1.CreateBuildRequest) (*BuildResult, error) {
       // Implementation
   }
   ```

4. **Register executor**:
   ```go
   // internal/executor/registry.go
   registry.Register("git", NewGitExecutor(logger, config))
   ```

### Adding Metrics

1. **Define metric**:
   ```go
   // internal/observability/metrics.go
   gitClonesDuration := prometheus.NewHistogramVec(
       prometheus.HistogramOpts{
           Name: "builderd_git_clone_duration_seconds",
           Help: "Git clone operation duration",
       },
       []string{"tenant_id", "success"},
   )
   ```

2. **Register metric**:
   ```go
   prometheus.MustRegister(gitClonesDuration)
   ```

3. **Record metric**:
   ```go
   timer := prometheus.NewTimer(gitClonesDuration.WithLabelValues(
       tenantID, "true"))
   defer timer.ObserveDuration()
   ```

## Troubleshooting Development Issues

### Common Problems

1. **Module Dependencies**:
   ```bash
   # Clear module cache
   go clean -modcache
   
   # Re-download dependencies
   go mod download
   ```

2. **Build Errors**:
   ```bash
   # Clean build cache
   go clean -cache
   
   # Rebuild
   go build -a ./cmd/builderd
   ```

3. **Test Failures**:
   ```bash
   # Run specific test with verbose output
   go test -v -run TestBuilderService ./internal/service/
   ```

### Development FAQ

**Q: How do I test without Docker?**
A: Set up mock executors that simulate Docker operations without actual containers.

**Q: How do I debug tenant isolation?**
A: Use debug logging and inspect namespace/cgroup creation in `/sys/fs/cgroup/`.

**Q: How do I test with different storage backends?**
A: Use environment variables to switch between local/S3/GCS backends.

## Resources

- [Go Documentation](https://go.dev/doc/)
- [ConnectRPC Documentation](https://connect.build/docs/go/)
- [Docker SDK Documentation](https://pkg.go.dev/github.com/docker/docker/client)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)