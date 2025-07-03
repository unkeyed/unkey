# Assetmanagerd Development Guide

This document provides comprehensive guidance for developing, testing, and contributing to assetmanagerd.

## Development Environment Setup

### Prerequisites

**Required Tools**:
- Go 1.24.4 or later
- Protocol Buffers compiler (`protoc`)
- SQLite 3.x
- Make
- Docker (optional, for integration testing)

**Go Dependencies**: [go.mod](../../go.mod)

### Local Development

#### Clone and Build

```bash
# Clone the repository
cd /path/to/unkey/go/deploy/assetmanagerd

# Install dependencies
go mod download

# Generate protobuf code
make generate

# Build the binary
make build

# Run tests
make test
```

#### Makefile Targets

**Build System**: [Makefile](../../Makefile)

```bash
# Development commands
make build          # Build binary
make test           # Run unit tests
make lint           # Run linters
make generate       # Generate protobuf code
make clean          # Clean build artifacts

# Installation commands
make install        # Install with systemd unit
make uninstall      # Remove systemd unit
```

### IDE Configuration

#### VS Code Settings

Create `.vscode/settings.json`:

```json
{
  "go.buildTags": "integration",
  "go.testFlags": ["-v", "-race"],
  "go.lintTool": "golangci-lint",
  "protobuf.path": [
    "proto",
    "../builderd/proto"
  ]
}
```

#### GoLand Configuration

- Enable Go modules support
- Configure Protocol Buffers plugin
- Set build tags for integration tests
- Enable race detection for tests

## Code Architecture

### Project Structure

```
assetmanagerd/
├── cmd/
│   ├── assetmanagerd/          # Main service binary
│   └── assetmanagerd-cli/      # CLI client
├── internal/
│   ├── builderd/               # Builderd client integration
│   ├── config/                 # Configuration management
│   ├── observability/          # Telemetry and monitoring
│   ├── registry/               # SQLite asset registry
│   ├── service/                # gRPC service implementation
│   └── storage/                # Storage backend interfaces
├── client/                     # Go client library
├── proto/asset/v1/             # Protocol buffer definitions
├── gen/asset/v1/              # Generated code
├── contrib/
│   ├── grafana-dashboards/    # Monitoring dashboards
│   └── systemd/               # Service unit files
└── docs/                      # Documentation
```

### Design Principles

#### AIDEV Anchors

The codebase uses AIDEV anchor comments for important design decisions:

**Search Command**: `grep -r "AIDEV-" assetmanagerd/`

Key anchors to understand:
- `AIDEV-NOTE`: Implementation explanations
- `AIDEV-BUSINESS_RULE`: Critical business logic
- `AIDEV-TODO`: Planned improvements
- `AIDEV-QUESTION`: Areas needing clarification

#### Error Handling

Consistent error handling patterns:

```go
// Use ConnectRPC error codes
return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))

// Wrap storage errors
if err != nil {
    return fmt.Errorf("failed to store asset: %w", err)
}

// Log errors with context
s.logger.LogAttrs(ctx, slog.LevelError, "failed to get asset",
    slog.String("id", assetID),
    slog.String("error", err.Error()),
)
```

#### Context Propagation

Proper context usage throughout the codebase:

```go
// Always pass context from RPC handlers
func (s *Service) RegisterAsset(ctx context.Context, req *connect.Request[...]) {
    // Pass context to all operations
    exists, err := s.storage.Exists(ctx, location)
    
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-resultChan:
        return result
    }
}
```

## Testing Strategy

### Unit Tests

Comprehensive unit test coverage with mocks and table-driven tests.

#### Test Structure

```go
func TestService_RegisterAsset(t *testing.T) {
    tests := []struct {
        name           string
        request        *assetv1.RegisterAssetRequest
        mockSetup      func(*MockStorage, *MockRegistry)
        expectedError  string
        expectedAsset  *assetv1.Asset
    }{
        {
            name: "successful registration",
            request: &assetv1.RegisterAssetRequest{
                Name:     "test-asset",
                Type:     assetv1.AssetType_ASSET_TYPE_KERNEL,
                Location: "test/path",
            },
            mockSetup: func(storage *MockStorage, registry *MockRegistry) {
                storage.EXPECT().Exists(gomock.Any(), "test/path").Return(true, nil)
                registry.EXPECT().CreateAsset(gomock.Any()).Return(nil)
            },
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### Mock Generation

Generate mocks for interfaces:

```bash
//go:generate mockgen -source=internal/storage/storage.go -destination=mocks/storage_mock.go
//go:generate mockgen -source=internal/registry/registry.go -destination=mocks/registry_mock.go
```

### Integration Tests

Full-stack integration tests with real dependencies.

#### Test Environment

```go
func setupTestEnvironment(t *testing.T) (*Service, func()) {
    // Create temporary directories
    tempDir := t.TempDir()
    
    // Setup test database
    dbPath := filepath.Join(tempDir, "test.db")
    registry, err := registry.New(dbPath, slog.Default())
    require.NoError(t, err)
    
    // Setup test storage
    storagePath := filepath.Join(tempDir, "storage")
    storage, err := storage.NewLocalBackend(storagePath, slog.Default())
    require.NoError(t, err)
    
    // Create service
    cfg := &config.Config{
        StorageBackend:   "local",
        LocalStoragePath: storagePath,
        DatabasePath:     dbPath,
    }
    
    service := service.New(cfg, slog.Default(), registry, storage, nil)
    
    return service, func() {
        registry.Close()
    }
}
```

#### Integration Test Examples

```go
func TestIntegration_AssetLifecycle(t *testing.T) {
    service, cleanup := setupTestEnvironment(t)
    defer cleanup()
    
    // Test full asset lifecycle
    ctx := context.Background()
    
    // 1. Register asset
    registerReq := &assetv1.RegisterAssetRequest{...}
    registerResp, err := service.RegisterAsset(ctx, connect.NewRequest(registerReq))
    require.NoError(t, err)
    
    // 2. Acquire asset
    acquireReq := &assetv1.AcquireAssetRequest{...}
    acquireResp, err := service.AcquireAsset(ctx, connect.NewRequest(acquireReq))
    require.NoError(t, err)
    
    // 3. Release asset
    releaseReq := &assetv1.ReleaseAssetRequest{...}
    _, err = service.ReleaseAsset(ctx, connect.NewRequest(releaseReq))
    require.NoError(t, err)
    
    // 4. Delete asset
    deleteReq := &assetv1.DeleteAssetRequest{...}
    deleteResp, err := service.DeleteAsset(ctx, connect.NewRequest(deleteReq))
    require.NoError(t, err)
    require.True(t, deleteResp.Msg.Deleted)
}
```

### Performance Tests

Benchmarks for critical operations:

```go
func BenchmarkService_ListAssets(b *testing.B) {
    service, cleanup := setupBenchEnvironment(b)
    defer cleanup()
    
    // Pre-populate with test data
    populateTestAssets(b, service, 1000)
    
    ctx := context.Background()
    req := &assetv1.ListAssetsRequest{PageSize: 100}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.ListAssets(ctx, connect.NewRequest(req))
        require.NoError(b, err)
    }
}
```

### Test Data Management

#### Test Fixtures

```go
// test/fixtures/assets.go
func NewTestKernel() *assetv1.Asset {
    return &assetv1.Asset{
        Id:       "01HN123456789ABCDEF",
        Name:     "test-kernel",
        Type:     assetv1.AssetType_ASSET_TYPE_KERNEL,
        Status:   assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
        Backend:  assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
        Location: "test/vmlinux",
        Labels: map[string]string{
            "arch":    "x86_64",
            "version": "5.10",
        },
    }
}
```

#### Database Migrations

Test database schema evolution:

```go
func TestDatabase_Migration(t *testing.T) {
    // Test migration from older schema versions
    // Ensure backward compatibility
    // Verify data integrity after migration
}
```

## Protocol Buffer Development

### Schema Evolution

Safe protobuf schema changes:

#### Safe Changes
- Adding new fields (with default values)
- Adding new enum values (except for first position)
- Adding new RPCs to services
- Deprecating fields (don't remove immediately)

#### Breaking Changes (Avoid)
- Removing fields or changing field numbers
- Changing field types
- Removing enum values
- Changing RPC signatures

#### Example Evolution

```protobuf
// Before
message Asset {
  string id = 1;
  string name = 2;
  AssetType type = 3;
}

// After (safe evolution)
message Asset {
  string id = 1;
  string name = 2;
  AssetType type = 3;
  string description = 4;      // New optional field
  repeated string tags = 5;    // New repeated field
  string deprecated_field = 6 [deprecated = true];  // Deprecated safely
}
```

### Code Generation

Automated protobuf code generation:

```bash
# Generate Go code
buf generate

# Verify generated code
buf lint proto/
buf breaking proto/ --against 'https://github.com/unkeyed/unkey.git#branch=main,subdir=go/deploy/assetmanagerd/proto'
```

**Generation Config**: [buf.gen.yaml](../../buf.gen.yaml)

## Local Development Workflows

### Running Locally

#### Minimal Setup

```bash
# Set required environment variables
export UNKEY_ASSETMANAGERD_PORT=8083
export UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/tmp/vm-assets
export UNKEY_ASSETMANAGERD_DATABASE_PATH=/tmp/assets.db
export UNKEY_ASSETMANAGERD_TLS_MODE=disabled  # For local dev only

# Create storage directory
mkdir -p /tmp/vm-assets

# Run the service
go run cmd/assetmanagerd/main.go
```

#### With SPIFFE (Production-like)

```bash
# Start SPIRE agent (if not running)
sudo systemctl start spire-agent

# Use SPIFFE mode
export UNKEY_ASSETMANAGERD_TLS_MODE=spiffe
export UNKEY_ASSETMANAGERD_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock

# Run with mTLS
go run cmd/assetmanagerd/main.go
```

#### With Builderd Integration

```bash
# Start builderd first
cd ../builderd && go run cmd/builderd/main.go

# Enable builderd integration
export UNKEY_ASSETMANAGERD_BUILDERD_ENABLED=true
export UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT=https://localhost:8082

# Run assetmanagerd
go run cmd/assetmanagerd/main.go
```

### Development Tools

#### CLI Client

Test service functionality with the CLI client:

**CLI Implementation**: [assetmanagerd-cli/main.go](../../cmd/assetmanagerd-cli/main.go)

```bash
# Build CLI
go build -o assetmanagerd-cli cmd/assetmanagerd-cli/main.go

# List assets
./assetmanagerd-cli list

# Register an asset
./assetmanagerd-cli register --name test-kernel --type kernel --location /path/to/vmlinux

# Query with auto-build
./assetmanagerd-cli query --type rootfs --label docker_image=nginx:latest --auto-build
```

#### gRPC Client

Direct gRPC testing with grpcurl:

```bash
# List available services
grpcurl -plaintext localhost:8083 list

# Call specific RPC
grpcurl -plaintext -d '{"type": 1}' localhost:8083 asset.v1.AssetManagerService/ListAssets

# With mTLS (production)
grpcurl -cert client.crt -key client.key -cacert ca.crt localhost:8083 list
```

#### Database Inspection

Direct SQLite access for debugging:

```bash
# Open database
sqlite3 /tmp/assets.db

# Common queries
.schema                              # Show schema
SELECT COUNT(*) FROM assets;         # Asset count
SELECT * FROM assets LIMIT 5;       # Recent assets
SELECT * FROM asset_leases;          # Active leases
```

## Debugging Techniques

### Logging Configuration

Enhanced logging for development:

```go
// Enable debug logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Add request tracing
logger = logger.With(
    slog.String("request_id", requestID),
    slog.String("tenant_id", tenantID),
)
```

### Profiling

Go runtime profiling for performance analysis:

```go
import _ "net/http/pprof"

// Add pprof endpoint (development only)
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiles:
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine analysis
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Tracing

OpenTelemetry tracing for request flow analysis:

```go
// Add custom spans
ctx, span := otel.Tracer("assetmanagerd").Start(ctx, "custom_operation")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("asset.id", assetID),
    attribute.Int64("asset.size", size),
)
```

## Contributing Guidelines

### Code Style

#### Go Conventions
- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable names
- Write self-documenting code
- Add comments for exported functions

#### Linting Rules

**Linter Configuration**: [.golangci.yml](../../.golangci.yml)

```bash
# Run all linters
golangci-lint run

# Fix auto-fixable issues
golangci-lint run --fix
```

### Commit Guidelines

#### Commit Message Format

```
type(scope): description

[optional body]

[optional footer]
```

Examples:
```
feat(api): add streaming upload support for large assets
fix(storage): handle race condition in concurrent asset access
docs(api): update RegisterAsset RPC documentation
test(integration): add builderd integration test suite
```

#### Commit Types
- `feat`: New features
- `fix`: Bug fixes  
- `docs`: Documentation changes
- `test`: Test additions/modifications
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

### Pull Request Process

1. **Branch Creation**: Create feature branch from `main`
2. **Development**: Implement changes with tests
3. **Testing**: Ensure all tests pass
4. **Documentation**: Update relevant documentation
5. **Review**: Submit PR with clear description
6. **CI/CD**: Ensure all checks pass

#### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes without approval
```

### Release Process

#### Version Management

Semantic versioning with automated releases:

```bash
# Update version
git tag v0.3.0

# Build release
make release

# Publish artifacts
make publish
```

#### Release Notes

Automated generation from conventional commits:

```bash
# Generate changelog
conventional-changelog -p angular -i CHANGELOG.md -s
```

## Advanced Development

### Custom Storage Backends

Implementing new storage backends:

```go
// Implement the Backend interface
type S3Backend struct {
    client *s3.Client
    bucket string
}

func (b *S3Backend) Store(ctx context.Context, id string, reader io.Reader, size int64) (string, error) {
    // S3 upload implementation
}

// Register in storage factory
func NewBackend(cfg *config.Config, logger *slog.Logger) (Backend, error) {
    switch cfg.StorageBackend {
    case "s3":
        return NewS3Backend(cfg, logger)
    // ... other backends
    }
}
```

### Extension Points

#### Middleware

Add custom middleware for gRPC interceptors:

```go
// Custom interceptor
func customInterceptor() connect.UnaryInterceptorFunc {
    return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
        return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Custom logic before
            resp, err := next(ctx, req)
            // Custom logic after
            return resp, err
        })
    })
}
```

#### Event Hooks

Add event hooks for asset lifecycle:

```go
type AssetHook interface {
    OnAssetCreated(ctx context.Context, asset *assetv1.Asset) error
    OnAssetDeleted(ctx context.Context, asset *assetv1.Asset) error
}

// Register hooks in service
func (s *Service) RegisterHook(hook AssetHook) {
    s.hooks = append(s.hooks, hook)
}
```

### Performance Optimization

#### Database Optimization

```go
// Prepared statements for common queries
type PreparedQueries struct {
    getAsset     *sql.Stmt
    listAssets   *sql.Stmt
    createLease  *sql.Stmt
}

// Connection pooling tuning
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### Caching Layer

```go
// Asset metadata caching
type CachedRegistry struct {
    registry *Registry
    cache    *lru.Cache
    ttl      time.Duration
}

func (c *CachedRegistry) GetAsset(id string) (*assetv1.Asset, error) {
    // Check cache first
    if cached, ok := c.cache.Get(id); ok {
        return cached.(*assetv1.Asset), nil
    }
    
    // Fallback to database
    asset, err := c.registry.GetAsset(id)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    c.cache.Add(id, asset)
    return asset, nil
}
```