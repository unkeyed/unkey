# AssetManagerd Development Guide

This guide covers setting up a development environment, building, testing, and contributing to AssetManagerd.

## Development Environment

### Prerequisites

- **Go**: 1.21 or later
- **Make**: GNU Make 4.0+
- **SQLite**: 3.35+ (for CLI tools)
- **protoc**: Protocol Buffer compiler
- **buf**: For protobuf management
- **SPIRE**: For local mTLS testing (optional)

### Project Structure

```
assetmanagerd/
├── cmd/
│   └── assetmanagerd/
│       └── main.go              # Entry point
├── proto/
│   └── asset/v1/
│       └── asset.proto          # API definitions
├── gen/                         # Generated code (do not edit)
├── internal/
│   ├── config/                  # Configuration
│   ├── observability/           # Metrics and tracing
│   ├── registry/                # SQLite database
│   ├── service/                 # Business logic
│   └── storage/                 # Storage backends
├── build/                       # Build artifacts
├── contrib/
│   ├── systemd/                 # Service files
│   └── grafana-dashboards/      # Monitoring
├── Makefile                     # Build automation
└── go.mod                       # Go module definition
```

## Building

### Quick Start

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey
cd unkey/go/deploy/assetmanagerd

# Build the binary
make build

# Run locally
./build/assetmanagerd
```

### Makefile Targets

**Core targets** ([Makefile](../../Makefile)):

```bash
make build      # Build binary to ./build/
make install    # Build and install with systemd
make clean      # Remove build artifacts
make test       # Run unit tests
make lint       # Run linters
make proto      # Regenerate protobuf code
```

### Build Process

The build process ([Makefile:8-10](../../Makefile:8-10)):

```makefile
build:
	@echo "Building assetmanagerd..."
	go build -o build/assetmanagerd cmd/assetmanagerd/main.go
```

Version is embedded at build time ([cmd/assetmanagerd/main.go:21](../../cmd/assetmanagerd/main.go:21)):

```go
const version = "v0.1.0"
```

### Cross-Compilation

```bash
# Build for different platforms
GOOS=linux GOARCH=amd64 make build
GOOS=linux GOARCH=arm64 make build
```

## Local Development

### Running Locally

1. **Basic Setup**:
```bash
# Create directories
mkdir -p /tmp/assetmanagerd/{db,assets}

# Set minimal environment
export UNKEY_ASSETMANAGERD_DATABASE_PATH=/tmp/assetmanagerd/db/assets.db
export UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/tmp/assetmanagerd/assets
export UNKEY_ASSETMANAGERD_TLS_MODE=insecure  # For testing only!

# Run
./build/assetmanagerd
```

2. **With SPIFFE/SPIRE**:
```bash
# Start SPIRE agent (see SPIRE docs)
spire-agent run -config /path/to/agent.conf &

# Run with SPIFFE
export UNKEY_ASSETMANAGERD_TLS_MODE=spiffe
export SPIFFE_ENDPOINT_SOCKET=unix:///tmp/spire-agent/public/api.sock
./build/assetmanagerd
```

### Development Configuration

Create a `.env` file for development:

```bash
# .env.development
UNKEY_ASSETMANAGERD_PORT=8083
UNKEY_ASSETMANAGERD_METRICS_PORT=9467
UNKEY_ASSETMANAGERD_LOG_LEVEL=debug
UNKEY_ASSETMANAGERD_DATABASE_PATH=/tmp/assetmanagerd/assets.db
UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/tmp/assetmanagerd/assets
UNKEY_ASSETMANAGERD_GC_ENABLED=false  # Disable in dev
UNKEY_ASSETMANAGERD_TLS_MODE=insecure
```

Load environment:
```bash
source .env.development
./build/assetmanagerd
```

## Testing

### Unit Tests

Run the test suite:

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/registry/...

# Verbose output
go test -v ./...
```

### Integration Tests

Test against a running service:

```bash
# Start the service
./build/assetmanagerd &

# Run integration tests
go test -tags=integration ./tests/...

# Using grpcurl
grpcurl -plaintext localhost:8083 list
grpcurl -plaintext localhost:8083 asset.v1.AssetService/ListAssets
```

### Test Data

Generate test assets:

```bash
# Create test kernel
dd if=/dev/zero of=/tmp/test-kernel bs=1M count=50
sha256sum /tmp/test-kernel

# Register via grpcurl
grpcurl -plaintext -d '{
  "name": "test-kernel",
  "type": "ASSET_TYPE_KERNEL",
  "location": "/tmp/test-kernel",
  "size_bytes": 52428800,
  "checksum": "sha256:...",
  "labels": {"test": "true"}
}' localhost:8083 asset.v1.AssetService/RegisterAsset
```

## Protocol Buffers

### Modifying the API

1. **Edit the proto file**:
```bash
vim proto/asset/v1/asset.proto
```

2. **Regenerate code**:
```bash
make proto
```

3. **Update implementation**:
- Modify service methods in `internal/service/service.go`
- Update any affected clients

### Code Generation

The project uses [buf](https://buf.build/) for protobuf management:

```yaml
# buf.gen.yaml
version: v1
plugins:
  - plugin: go
    out: gen
    opt: paths=source_relative
  - plugin: connect-go
    out: gen
    opt: paths=source_relative
```

## Code Style

### Go Standards

Follow standard Go conventions:
- Run `gofmt` on all code
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable names
- Add comments for exported functions

### Project Conventions

1. **Error Handling**:
```go
if err != nil {
    return nil, fmt.Errorf("failed to create asset: %w", err)
}
```

2. **Logging**:
```go
slog.Info("asset registered",
    "asset_id", asset.ID,
    "type", asset.Type,
    "size_bytes", asset.SizeBytes,
)
```

3. **Context Usage**:
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Linting

Run linters before committing:

```bash
# golangci-lint (install: https://golangci-lint.run/)
golangci-lint run

# go vet
go vet ./...

# staticcheck
staticcheck ./...
```

## Debugging

### Enable Debug Logging

```bash
export UNKEY_ASSETMANAGERD_LOG_LEVEL=debug
./build/assetmanagerd
```

### Using Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the service
dlv debug ./cmd/assetmanagerd/main.go

# Set breakpoints
(dlv) break main.main
(dlv) break service.go:103  # RegisterAsset
(dlv) continue
```

### Performance Profiling

1. **Enable pprof** (add to main.go):
```go
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

2. **Collect profiles**:
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Database Debugging

```bash
# Open SQLite CLI
sqlite3 /tmp/assetmanagerd/assets.db

# Useful queries
.tables
.schema assets
SELECT * FROM assets;
SELECT COUNT(*) FROM asset_leases WHERE released_at IS NULL;

# Monitor in real-time
watch -n 1 'sqlite3 /tmp/assetmanagerd/assets.db "SELECT COUNT(*) FROM assets"'
```

## Adding Features

### Example: Adding a New Storage Backend

1. **Define the interface** ([internal/storage/storage.go](../../internal/storage/storage.go)):
```go
type Storage interface {
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Put(ctx context.Context, key string, reader io.Reader) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Copy(ctx context.Context, srcKey, dstPath string) error
}
```

2. **Implement the backend**:
```go
// internal/storage/s3.go
type S3Storage struct {
    client *s3.Client
    bucket string
}

func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
    // Implementation
}
```

3. **Register in factory**:
```go
// internal/storage/factory.go
func NewStorage(backend string, config Config) (Storage, error) {
    switch backend {
    case "s3":
        return NewS3Storage(config)
    // ...
    }
}
```

### Example: Adding a New RPC

1. **Update proto**:
```protobuf
rpc VerifyAsset(VerifyAssetRequest) returns (VerifyAssetResponse);

message VerifyAssetRequest {
    string asset_id = 1;
}

message VerifyAssetResponse {
    bool valid = 1;
    string actual_checksum = 2;
}
```

2. **Regenerate code**:
```bash
make proto
```

3. **Implement method**:
```go
func (s *Service) VerifyAsset(
    ctx context.Context,
    req *assetv1.VerifyAssetRequest,
) (*assetv1.VerifyAssetResponse, error) {
    // Implementation
}
```

## Contributing

### Workflow

1. **Fork and clone** the repository
2. **Create a feature branch**: `git checkout -b feature/my-feature`
3. **Make changes** and add tests
4. **Run tests**: `make test lint`
5. **Commit** with descriptive message
6. **Push** and create pull request

### Commit Messages

Follow conventional commits:
```
feat: add S3 storage backend
fix: prevent race condition in reference counting
docs: update API examples
test: add integration tests for GC
chore: update dependencies
```

### Pull Request Guidelines

- Include tests for new features
- Update documentation
- Ensure CI passes
- Add AIDEV comments for complex code
- Follow existing patterns

## Release Process

1. **Update version** in `main.go`
2. **Update CHANGELOG**
3. **Tag release**: `git tag v0.2.0`
4. **Build release binaries**:
```bash
# Build for multiple platforms
for GOOS in linux darwin; do
    for GOARCH in amd64 arm64; do
        GOOS=$GOOS GOARCH=$GOARCH make build
        mv build/assetmanagerd build/assetmanagerd-$GOOS-$GOARCH
    done
done
```

## Troubleshooting Development Issues

### Common Problems

**1. "cannot find module"**
```bash
go mod download
go mod tidy
```

**2. "proto files not found"**
```bash
make proto
```

**3. "permission denied" on storage**
```bash
# Check permissions
ls -la /tmp/assetmanagerd/
# Fix if needed
chmod -R 755 /tmp/assetmanagerd/
```

**4. Port already in use**
```bash
# Find process
lsof -i :8083
# Or use different port
export UNKEY_ASSETMANAGERD_PORT=8084
```

### Getting Help

- Check existing issues on GitHub
- Ask in development chat/forum
- Review service logs with debug enabled
- Use `git grep` to find examples in codebase