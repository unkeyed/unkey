# Builderd Development Guide

This guide covers building, testing, and contributing to the builderd service.

## Development Environment

### Prerequisites

**Required Software**:
- **Go**: 1.24.4+ (as specified in go.mod)
- **Docker**: 20.10+ for image extraction testing
- **Make**: For build automation
- **Git**: For version control
- **protoc**: Protocol buffer compiler (for proto changes)

**System Requirements**:
- **OS**: Linux (required for Docker integration)
- **Memory**: 8GB+ for development and testing
- **Storage**: 20GB+ free space for build artifacts and Docker images

### Getting Started

**Clone Repository**:
```bash
git clone https://github.com/unkeyed/unkey
cd go/deploy/builderd
```

**Install Dependencies**:
```bash
# Download Go dependencies
go mod download

# Verify Docker access
docker info
```

**Build from Source**:
```bash
# Build binary
make build

# Binary location
ls -la build/builderd
```

**Build Targets**: [Makefile](../../Makefile)

### Project Structure

```
builderd/
├── cmd/
│   ├── builderd/           # Main service binary
│   └── builderd-cli/       # CLI client tool
├── client/                 # Go client library
├── internal/
│   ├── assetmanager/      # Assetmanagerd client integration
│   ├── assets/            # Base asset management
│   ├── config/            # Configuration management
│   ├── executor/          # Build executor implementations
│   ├── observability/     # Metrics and tracing
│   ├── service/           # Main service implementation
│   ├── storage/           # Storage backend interfaces
│   └── tenant/            # Multi-tenant isolation
├── proto/
│   └── builder/v1/        # Protocol buffer definitions
├── gen/                   # Generated protobuf code
├── contrib/
│   ├── systemd/           # Systemd service files
│   └── grafana-dashboards/ # Monitoring dashboards
└── docs/                  # Service documentation
```

### Configuration for Development

**Development Environment**:
```bash
# Create development config
cat > environment.dev <<EOF
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_ADDRESS=127.0.0.1
UNKEY_BUILDERD_TLS_MODE=disabled
UNKEY_BUILDERD_ASSETMANAGER_ENABLED=false
UNKEY_BUILDERD_OTEL_ENABLED=true
UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED=false
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/tmp/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/tmp/builderd/workspace
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd/scratch
EOF

# Source configuration
source environment.dev

# Create required directories
mkdir -p /tmp/builderd/{rootfs,workspace,scratch}
```

**Run Development Server**:
```bash
# Run with configuration
./build/builderd

# Or with live reload (requires air)
air
```

### Code Generation

**Protocol Buffers**:
```bash
# Regenerate protobuf code
make generate

# Verify generated files
ls -la gen/builder/v1/
```

**Generation Dependencies**:
- `protoc-gen-go`: Go protobuf plugin
- `protoc-gen-connect-go`: ConnectRPC plugin

**Generation Configuration**: [buf.gen.yaml](../../buf.gen.yaml)

## Testing

### Unit Tests

**Run All Tests**:
```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Verbose test output
go test -v ./...
```

**Test Patterns**:
```bash
# Test specific package
go test ./internal/executor/

# Test with build tags
go test -tags integration ./...

# Run specific test
go test -run TestDockerExecutor_Extract ./internal/executor/
```

### Integration Tests

**Prerequisites**:
```bash
# Ensure Docker is running
systemctl start docker

# Pull test images
docker pull alpine:latest
docker pull nginx:latest
```

**Run Integration Tests**:
```bash
# Integration tests (requires Docker)
make test-integration

# End-to-end tests
make test-e2e
```

### Testing Patterns

**Mock Generation**:
```bash
# Generate mocks for interfaces
go generate ./...

# Mock locations
ls -la internal/*/mocks/
```

**Test Fixtures**:
```bash
# Test data location
testdata/
├── docker-images/     # Sample Docker configurations
├── build-configs/     # Sample build configurations
└── expected-outputs/  # Expected test results
```

**Example Test Structure**:
```go
func TestDockerExecutor_ExtractImage(t *testing.T) {
    // Arrange
    executor := NewDockerExecutor(logger, config, metrics)
    request := &builderv1.CreateBuildRequest{
        Config: &builderv1.BuildConfig{
            Source: &builderv1.BuildSource{
                SourceType: &builderv1.BuildSource_DockerImage{
                    DockerImage: &builderv1.DockerImageSource{
                        ImageUri: "alpine:latest",
                    },
                },
            },
        },
    }

    // Act
    result, err := executor.Execute(ctx, request)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "completed", result.Status)
    assert.FileExists(t, result.RootfsPath)
}
```

### Benchmarking

**Performance Tests**:
```bash
# Run benchmarks
go test -bench=. ./internal/executor/

# Benchmark with memory profiling
go test -bench=. -memprofile=mem.prof ./internal/executor/

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/executor/
```

**Example Benchmark**:
```go
func BenchmarkDockerExtraction(b *testing.B) {
    executor := NewDockerExecutor(logger, config, nil)
    
    for i := 0; i < b.N; i++ {
        result, err := executor.Extract(ctx, request)
        require.NoError(b, err)
        cleanup(result.WorkspaceDir)
    }
}
```

## Development Workflow

### Code Style

**Go Format**:
```bash
# Format code
go fmt ./...

# Import organization
goimports -w .

# Linting
golangci-lint run
```

**Linting Configuration**: [.golangci.yml](../../.golangci.yml)

### Git Workflow

**Branch Naming**:
- `feature/description` - New features
- `fix/description` - Bug fixes  
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

**Commit Message Format**:
```
type(scope): description

feat(executor): add git repository support
fix(docker): handle large image extraction
docs(api): update build configuration examples
```

### Pre-commit Hooks

**Setup Pre-commit**:
```bash
# Install pre-commit hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

**Hook Configuration**: [.pre-commit-config.yaml](../../.pre-commit-config.yaml)

## Debugging

### Local Debugging

**VS Code Configuration**:
```json
{
    "name": "Debug builderd",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/cmd/builderd/main.go",
    "env": {
        "UNKEY_BUILDERD_PORT": "8082",
        "UNKEY_BUILDERD_TLS_MODE": "disabled",
        "UNKEY_BUILDERD_OTEL_ENABLED": "false"
    },
    "args": []
}
```

**Delve Debugging**:
```bash
# Debug with delve
dlv debug ./cmd/builderd/main.go

# Remote debugging
dlv debug --headless --listen=:2345 --api-version=2 ./cmd/builderd/main.go
```

### Trace Analysis

**OpenTelemetry Traces**:
```bash
# Enable tracing in development
export UNKEY_BUILDERD_OTEL_ENABLED=true
export UNKEY_BUILDERD_OTEL_ENDPOINT=http://localhost:4318

# Run with Jaeger
docker run -d -p 16686:16686 -p 14268:14268 jaegertracing/all-in-one:latest
```

**Trace Viewing**: Visit http://localhost:16686 for Jaeger UI

### Log Debugging

**Debug Logging**:
```bash
# Enable debug logs
export UNKEY_BUILDERD_LOG_LEVEL=debug

# Structured log analysis
./build/builderd 2>&1 | jq 'select(.level == "error")'
```

**Log Fields for Debugging**:
- `build_id` - Build job identifier
- `tenant_id` - Tenant context
- `source_type` - Build source type
- `error` - Error messages and stack traces

## Adding New Features

### Executor Pattern

**Creating New Executor**:
```go
// internal/executor/git.go
type GitExecutor struct {
    logger *slog.Logger
    config *config.Config
    metrics *observability.BuildMetrics
}

func NewGitExecutor(logger *slog.Logger, cfg *config.Config, metrics *observability.BuildMetrics) *GitExecutor {
    return &GitExecutor{
        logger:  logger,
        config:  cfg,
        metrics: metrics,
    }
}

func (g *GitExecutor) Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error) {
    // Implementation
}

func (g *GitExecutor) GetSupportedSources() []string {
    return []string{"git"}
}
```

**Register Executor**: [internal/executor/registry.go:36](../../internal/executor/registry.go#L36)

### Protocol Buffer Changes

**Updating Proto Definitions**:
1. Modify [proto/builder/v1/builder.proto](../../proto/builder/v1/builder.proto)
2. Run `make generate` to update generated code
3. Update service implementation to handle new fields
4. Add tests for new functionality
5. Update API documentation

**Example Proto Addition**:
```protobuf
// Add new build strategy
message PythonBuildStrategy {
  string python_version = 1;
  string requirements_file = 2;
  bool use_poetry = 3;
}

// Update BuildStrategy
message BuildStrategy {
  oneof strategy_type {
    DockerExtractStrategy docker_extract = 1;
    GitBuildStrategy git_build = 2;
    PythonBuildStrategy python_build = 3;  // New strategy
  }
}
```

### Configuration Changes

**Adding Configuration Options**:
1. Update config structs in [internal/config/config.go](../../internal/config/config.go)
2. Add environment variable parsing
3. Add validation in `validateConfig()`
4. Update documentation and examples
5. Add tests for new configuration

**Example Configuration Addition**:
```go
type GitConfig struct {
    Enabled        bool          `yaml:"enabled"`
    CloneTimeout   time.Duration `yaml:"clone_timeout"`
    MaxRepoSizeGB  int           `yaml:"max_repo_size_gb"`
    AllowedHosts   []string      `yaml:"allowed_hosts"`
}

// Add to main Config struct
type Config struct {
    // ... existing fields
    Git GitConfig `yaml:"git"`
}

// Add environment variable parsing
Git: GitConfig{
    Enabled:       getEnvBoolOrDefault("UNKEY_BUILDERD_GIT_ENABLED", false),
    CloneTimeout:  getEnvDurationOrDefault("UNKEY_BUILDERD_GIT_CLONE_TIMEOUT", 5*time.Minute),
    MaxRepoSizeGB: getEnvIntOrDefault("UNKEY_BUILDERD_GIT_MAX_REPO_SIZE_GB", 1),
    AllowedHosts:  getEnvSliceOrDefault("UNKEY_BUILDERD_GIT_ALLOWED_HOSTS", []string{"github.com", "gitlab.com"}),
},
```

## Contribution Guidelines

### Development Process

1. **Issue Creation**: Create GitHub issue describing the feature/bug
2. **Branch Creation**: Create feature branch from main
3. **Implementation**: Implement changes with tests
4. **Testing**: Ensure all tests pass and add new tests
5. **Documentation**: Update relevant documentation
6. **Pull Request**: Create PR with detailed description
7. **Review**: Address code review feedback
8. **Merge**: Squash and merge after approval

### Code Review Checklist

**Functionality**:
- [ ] Implementation matches requirements
- [ ] Error handling is comprehensive
- [ ] Resource cleanup is proper
- [ ] Logging includes relevant context

**Testing**:
- [ ] Unit tests cover new functionality
- [ ] Integration tests for external dependencies
- [ ] Edge cases are tested
- [ ] Performance impact is acceptable

**Code Quality**:
- [ ] Code follows Go conventions
- [ ] Functions are focused and testable
- [ ] Comments explain complex logic
- [ ] No security vulnerabilities introduced

**Documentation**:
- [ ] API documentation updated
- [ ] Configuration documented
- [ ] Examples provided where appropriate
- [ ] Breaking changes noted

### Release Process

**Version Management**:
- Follow semantic versioning (v0.x.y)
- Update version in relevant files
- Create release notes with changes
- Tag releases in Git

**Deployment Testing**:
- Test in staging environment
- Verify backward compatibility
- Check service integration points
- Validate configuration migration

## Architecture Decisions

### Design Principles

1. **Tenant Isolation**: All operations must respect tenant boundaries
2. **Observability First**: Comprehensive logging, metrics, and tracing
3. **Graceful Degradation**: Failures should not cascade across tenants
4. **Resource Efficiency**: Optimize for resource usage and cleanup
5. **Security by Default**: Secure configurations and input validation

### Technology Choices

**Why ConnectRPC**: Type-safe, HTTP/2-based RPC with excellent tooling
**Why SPIFFE/SPIRE**: Industry-standard workload identity for zero-trust
**Why OpenTelemetry**: Vendor-neutral observability with rich ecosystem
**Why Docker**: Ubiquitous container runtime with excellent tooling

### Future Considerations

**Planned Enhancements**:
- Database integration for persistent storage
- Build result caching for performance
- Multi-node deployment support
- Enhanced security scanning

**Technical Debt**:
- In-memory build storage (needs database)
- Limited executor types (expand beyond Docker)
- Basic tenant management (needs enhancement)

Source: [AIDEV comments throughout codebase](../../internal/service/builder.go)