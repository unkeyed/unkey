# builderd Development Guide

This guide covers building, testing, and developing builderd.

## Prerequisites

- **Go**: Version 1.21 or later
- **Make**: For build automation
- **Docker**: For testing Docker image builds
- **Protocol Buffers**: For regenerating protobuf code
- **SPIRE**: (Optional) For testing SPIFFE/mTLS

## Building

### Quick Build

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey.git
cd unkey/go/deploy/builderd

# Build the binary
make build

# Install with systemd unit
make install
```

### Build Targets

The Makefile provides several targets:

```bash
make build      # Build binary to build/builderd
make install    # Build and install with systemd
make test       # Run unit tests
make lint       # Run linters
make clean      # Clean build artifacts
make proto      # Regenerate protobuf code
```

### Build Flags

```bash
# Production build with version
make build VERSION=1.0.0

# Debug build
go build -gcflags="all=-N -l" -o build/builderd ./cmd/builderd

# Static binary
CGO_ENABLED=0 go build -ldflags="-w -s" -o build/builderd ./cmd/builderd
```

## Project Structure

```
builderd/
├── cmd/
│   └── builderd/          # Main entry point
│       └── main.go
├── internal/              # Private packages
│   ├── config/           # Configuration management
│   ├── executor/         # Build executors
│   ├── observability/    # Metrics and tracing
│   ├── service/          # RPC service implementation
│   └── tenant/           # Multi-tenancy
├── proto/                # Protocol buffer definitions
│   └── builder/
│       └── v1/
│           └── builder.proto
├── gen/                  # Generated code
│   └── proto/
│       └── builder/
│           └── v1/
├── docs/                 # Documentation
├── contrib/              # Additional resources
│   ├── systemd/         # Systemd unit files
│   └── grafana-dashboards/
├── build/               # Build output directory
├── Makefile
├── go.mod
└── go.sum
```

## Development Setup

### Local Environment

1. **Install dependencies**:
```bash
# Install Go
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/bin/go

# Install protoc
sudo apt-get install -y protobuf-compiler

# Install Go protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

2. **Setup workspace**:
```bash
# Create required directories
sudo mkdir -p /opt/builderd/{rootfs,workspace,data}
sudo chown -R $USER:$USER /opt/builderd
```

3. **Configure environment**:
```bash
# Create .env file for development
cat > .env <<EOF
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_STORAGE_BACKEND=local
UNKEY_BUILDERD_LOG_LEVEL=debug
UNKEY_BUILDERD_OTEL_ENABLED=false
UNKEY_BUILDERD_TLS_MODE=disabled
EOF

# Source environment
source .env
```

### Running Locally

```bash
# Run with default configuration
go run ./cmd/builderd

# Run with custom config
UNKEY_BUILDERD_PORT=8090 go run ./cmd/builderd

# Run with live reload (using air)
go install github.com/cosmtrek/air@latest
air
```

### Docker Development

```bash
# Build development image
docker build -t builderd:dev -f Dockerfile.dev .

# Run with volume mounts
docker run -it --rm \
  -v $(pwd):/workspace \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -p 8082:8082 \
  -p 9466:9466 \
  builderd:dev
```

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./internal/executor/...

# Run with race detection
go test -race ./...
```

### Integration Tests

```bash
# Run integration tests (requires Docker)
go test -tags=integration ./tests/integration/...

# Test with real registry
export BUILDERD_TEST_REGISTRY_AUTH="token"
go test -v ./internal/executor/docker_test.go
```

### Test Coverage Goals

- Unit test coverage: >80%
- Integration test coverage: >60%
- Critical paths: 100%

## Code Generation

### Regenerating Protobuf

```bash
# Update proto file
vim proto/builder/v1/builder.proto

# Regenerate Go code
make proto

# Or manually
protoc --go_out=gen --go_opt=paths=source_relative \
       --connect-go_out=gen --connect-go_opt=paths=source_relative \
       proto/builder/v1/builder.proto
```

### Adding New RPC Methods

1. Update proto definition:
```protobuf
service BuilderService {
  // ... existing methods ...
  
  // New method
  rpc GetBuildArtifacts(GetBuildArtifactsRequest) 
      returns (GetBuildArtifactsResponse);
}

message GetBuildArtifactsRequest {
  string build_id = 1;
  string tenant_id = 2;
}

message GetBuildArtifactsResponse {
  repeated Artifact artifacts = 1;
}
```

2. Regenerate code:
```bash
make proto
```

3. Implement handler:
```go
func (s *BuilderService) GetBuildArtifacts(
    ctx context.Context,
    req *connect.Request[builderv1.GetBuildArtifactsRequest],
) (*connect.Response[builderv1.GetBuildArtifactsResponse], error) {
    // Implementation
}
```

## Debugging

### Debug Logging

```go
// Add debug logs
s.logger.DebugContext(ctx, "processing build request",
    slog.String("build_id", buildID),
    slog.Any("config", req.Msg.Config),
)
```

### Remote Debugging

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o build/builderd ./cmd/builderd

# Run with Delve
dlv exec ./build/builderd

# Or attach to running process
dlv attach $(pgrep builderd)
```

### Performance Profiling

```go
// Add pprof endpoints
import _ "net/http/pprof"

// In main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Contributing

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports`
- Run linters before committing:
```bash
make lint
```

### Commit Guidelines

```bash
# Format: <type>(<scope>): <subject>
git commit -m "feat(executor): add git repository support"
git commit -m "fix(tenant): correct quota calculation"
git commit -m "docs(api): update build state descriptions"
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Maintenance

### Pull Request Process

1. Create feature branch:
```bash
git checkout -b feat/my-feature
```

2. Make changes and test:
```bash
make test
make lint
```

3. Update documentation if needed

4. Submit PR with description

### Adding Dependencies

```bash
# Add new dependency
go get github.com/some/package

# Update go.mod and go.sum
go mod tidy

# Verify
go mod verify
```

## Common Development Tasks

### Adding a New Executor

1. Create executor interface implementation:
```go
// internal/executor/git.go
type GitExecutor struct {
    logger *slog.Logger
    config *config.Config
}

func (e *GitExecutor) Execute(ctx context.Context, req *builderv1.CreateBuildRequest) (*ExecutorResult, error) {
    // Implementation
}
```

2. Register in registry:
```go
// internal/executor/registry.go
func NewRegistry() *Registry {
    r := &Registry{
        executors: make(map[string]Executor),
    }
    r.Register("git", NewGitExecutor())
    return r
}
```

### Adding Metrics

```go
// internal/observability/metrics.go
var (
    gitCloneCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "builderd_git_clones_total",
            Help: "Total number of git clones",
        },
        []string{"tenant", "status"},
    )
)

// Register metric
func init() {
    prometheus.MustRegister(gitCloneCounter)
}

// Use metric
gitCloneCounter.WithLabelValues(tenantID, "success").Inc()
```

### Testing with Mock Client

```go
// Create test client
client := builderv1connect.NewBuilderServiceClient(
    httptest.NewClient(mockHandler),
    "http://test",
)

// Make request
resp, err := client.CreateBuild(ctx, connect.NewRequest(&builderv1.CreateBuildRequest{
    Config: testConfig,
}))
```

## Release Process

1. **Update version**:
```bash
# Update version in main.go
sed -i 's/version = ".*"/version = "1.0.0"/' cmd/builderd/main.go
```

2. **Update changelog**:
```bash
# Add release notes
vim CHANGELOG.md
```

3. **Tag release**:
```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

4. **Build release artifacts**:
```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 make build
GOOS=linux GOARCH=arm64 make build
```

## Troubleshooting Development Issues

### Common Problems

**Module errors**:
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download
```

**Build errors**:
```bash
# Verify Go version
go version

# Check for missing tools
which protoc
which protoc-gen-go
```

**Test failures**:
```bash
# Run tests verbosely
go test -v -run TestName ./...

# Check test dependencies
docker ps  # Ensure Docker is running
```

### Getting Help

- Check existing issues: [GitHub Issues](https://github.com/unkeyed/unkey/issues)
- Review documentation: [docs/](../README.md)
- Ask questions: Create a new issue with the `question` label

AIDEV-NOTE: Always ensure code changes maintain backward compatibility and include appropriate tests.