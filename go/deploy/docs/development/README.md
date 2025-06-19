# Unkey Deploy Development Guide

## Development Environment Setup

### Prerequisites

- Go 1.23 or later
- Make
- Git
- Docker (optional, for integration testing)
- systemd (for local service testing)

### Initial Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/unkeyed/unkey.git
   cd unkey/go/deploy
   ```

2. **Install development dependencies**:
   ```bash
   # Install Go tools
   go install github.com/bufbuild/buf/cmd/buf@latest
   go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   
   # Install linting tools
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

3. **Build all services**:
   ```bash
   make build
   ```

## Service Standards

### Code Organization

Each service follows this structure:
```
service/
├── cmd/
│   └── service/          # Main binary
│       └── main.go
├── internal/             # Private packages
│   ├── api/             # API handlers
│   ├── config/          # Configuration
│   ├── service/         # Business logic
│   └── storage/         # Data access
├── pkg/                 # Public packages
├── proto/               # Protocol buffers
├── contrib/             # Additional resources
│   ├── systemd/         # Service units
│   └── grafana/         # Dashboards
├── Makefile             # Service build
└── README.md            # Service docs
```

### Coding Standards

1. **Error Handling**:
   ```go
   // Always wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to create VM %s: %w", vmID, err)
   }
   ```

2. **Logging**:
   ```go
   // Use structured logging
   log.Info("VM created",
       "vm_id", vmID,
       "customer_id", customerID,
       "duration_ms", time.Since(start).Milliseconds(),
   )
   ```

3. **Metrics**:
   ```go
   // Increment counters
   metricsCollector.VMsCreatedTotal.Inc()
   
   // Record histograms
   metricsCollector.APIRequestDuration.WithLabelValues("CreateVm").Observe(duration.Seconds())
   ```

4. **Context Usage**:
   ```go
   // Always accept and propagate context
   func (s *Service) CreateVM(ctx context.Context, req *pb.CreateVMRequest) (*pb.CreateVMResponse, error) {
       // Use context for cancellation and tracing
       span, ctx := opentelemetry.StartSpan(ctx, "CreateVM")
       defer span.End()
   }
   ```

### Environment Variables

Follow the naming convention:
```bash
UNKEY_<SERVICE>_<COMPONENT>_<VARIABLE>

# Examples:
UNKEY_METALD_API_PORT=8080
UNKEY_BILLAGED_STORAGE_PATH=/var/lib/billaged
UNKEY_ASSETMANAGERD_CACHE_SIZE_GB=50
```

## API Design Guidelines

### Protocol Buffers

1. **Versioning**:
   ```protobuf
   syntax = "proto3";
   package unkey.deploy.metald.v1;
   ```

2. **Request/Response Naming**:
   ```protobuf
   service MetaldService {
     rpc CreateVm(CreateVmRequest) returns (CreateVmResponse);
   }
   
   message CreateVmRequest {
     string customer_id = 1;
     VmConfig config = 2;
   }
   
   message CreateVmResponse {
     string vm_id = 1;
     VmStatus status = 2;
   }
   ```

3. **Field Guidelines**:
   - Use snake_case for field names
   - Add field comments for documentation
   - Reserve fields instead of deleting
   - Use well-known types (Timestamp, Duration)

### API Patterns

1. **List Operations**:
   ```protobuf
   message ListVmsRequest {
     int32 page_size = 1;
     string page_token = 2;
     string filter = 3;  // SQL-like filter
   }
   ```

2. **Long-Running Operations**:
   ```protobuf
   message Operation {
     string id = 1;
     enum Status {
       PENDING = 0;
       RUNNING = 1;
       SUCCEEDED = 2;
       FAILED = 3;
     }
     Status status = 2;
     google.protobuf.Any result = 3;
   }
   ```

3. **Error Handling**:
   ```go
   // Use Connect error codes
   if customer == "" {
       return nil, connect.NewError(connect.CodeInvalidArgument, 
           errors.New("customer_id is required"))
   }
   ```

## Testing Strategies

### Unit Testing

```go
func TestVMCreation(t *testing.T) {
    // Arrange
    service := NewService(MockStorage(), MockBilling())
    req := &pb.CreateVmRequest{
        CustomerId: "cust_123",
        Config: &pb.VmConfig{
            Memory: 512,
            Vcpus: 1,
        },
    }
    
    // Act
    resp, err := service.CreateVm(context.Background(), req)
    
    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, resp.VmId)
    assert.Equal(t, pb.VmStatus_CREATING, resp.Status)
}
```

### Integration Testing

```go
func TestServiceIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Start real services
    ctx := context.Background()
    assetMgr := startAssetManager(t)
    billingSvc := startBillingService(t)
    metald := startMetald(t, assetMgr.Addr(), billingSvc.Addr())
    
    // Test full flow
    client := metald.Client()
    // ... test VM lifecycle
}
```

### Contract Testing

Ensure service contracts match:
```bash
# Generate mocks from protobufs
buf generate

# Run contract tests
go test ./internal/contracts/...
```

### Load Testing

```go
func BenchmarkCreateVM(b *testing.B) {
    service := NewService(...)
    req := &pb.CreateVmRequest{...}
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.CreateVm(context.Background(), req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Development Workflow

### Local Development

1. **Run services locally**:
   ```bash
   # Terminal 1: assetmanagerd
   UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local \
   UNKEY_ASSETMANAGERD_STORAGE_LOCAL_PATH=/tmp/assets \
   ./assetmanagerd/build/assetmanagerd
   
   # Terminal 2: billaged
   ./billaged/build/billaged
   
   # Terminal 3: metald (requires root for Firecracker)
   sudo UNKEY_METALD_MOCK_MODE=true \
   ./metald/build/metald
   ```

2. **Use mock mode for testing**:
   ```bash
   UNKEY_METALD_MOCK_MODE=true
   UNKEY_METALD_MOCK_DELAY_MS=100
   ```

3. **Enable debug logging**:
   ```bash
   UNKEY_*_LOG_LEVEL=debug
   UNKEY_*_LOG_FORMAT=text  # Better for development
   ```

### Making Changes

1. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature
   ```

2. **Make changes and test**:
   ```bash
   # Run tests
   make test
   
   # Run linter
   make lint
   
   # Build service
   SERVICE=metald make build
   ```

3. **Update version for significant changes**:
   ```go
   // internal/version/version.go
   const Version = "0.2.1"  // Increment patch version
   ```

4. **Add AIDEV markers for complex code**:
   ```go
   // AIDEV-NOTE: This implements gap detection for billing data.
   // It tracks the last seen timestamp per VM and triggers recovery
   // when gaps exceed the threshold.
   func (s *Service) detectGaps() error {
       // AIDEV-TODO: Add exponential backoff for recovery attempts
   }
   ```

### Debugging

1. **Enable detailed logging**:
   ```bash
   UNKEY_*_LOG_LEVEL=trace
   UNKEY_*_LOG_CALLER=true
   ```

2. **Use debug endpoints**:
   ```bash
   # Dump service state
   curl http://localhost:8080/debug/state
   
   # Profile CPU
   go tool pprof http://localhost:9464/debug/pprof/profile
   ```

3. **Trace requests**:
   ```bash
   # Add trace header
   curl -H "X-Trace-Id: test-123" http://localhost:8080/api/v1/vms
   
   # Find in logs
   journalctl -u metald | grep test-123
   ```

## Contributing Guidelines

### Pull Request Process

1. **Before submitting**:
   - Run `make lint` - no warnings allowed
   - Run `make test` - all tests must pass
   - Update documentation if needed
   - Add tests for new functionality

2. **PR description template**:
   ```markdown
   ## Summary
   Brief description of changes
   
   ## Testing
   - [ ] Unit tests added/updated
   - [ ] Integration tests pass
   - [ ] Manual testing completed
   
   ## Documentation
   - [ ] API docs updated
   - [ ] README updated if needed
   - [ ] AIDEV markers added for complex logic
   ```

3. **Review criteria**:
   - Code follows service standards
   - Errors are properly wrapped
   - Metrics and logging added
   - No security vulnerabilities
   - Performance impact considered

### Documentation Requirements

1. **Code comments**:
   ```go
   // CreateVM creates a new Firecracker microVM with the specified configuration.
   // It validates the request, prepares assets, configures networking, and starts
   // the VM. Returns an error if any step fails.
   func (s *Service) CreateVM(ctx context.Context, req *pb.CreateVmRequest) (*pb.CreateVmResponse, error) {
   ```

2. **AIDEV markers**:
   ```go
   // AIDEV-BUSINESS_RULE: VMs are billed per-second with a minimum of 10 seconds.
   // This prevents rapid create/destroy cycles from avoiding charges.
   
   // AIDEV-QUESTION: Should we implement VM hibernation for cost savings?
   ```

3. **README updates**:
   - Update service README for new features
   - Add examples for new APIs
   - Document new configuration options

## Common Development Tasks

### Adding a New RPC Method

1. **Update protobuf**:
   ```protobuf
   service MetaldService {
     rpc ResizeVm(ResizeVmRequest) returns (ResizeVmResponse);
   }
   ```

2. **Generate code**:
   ```bash
   cd metald && buf generate
   ```

3. **Implement handler**:
   ```go
   func (s *Server) ResizeVm(ctx context.Context, req *connect.Request[pb.ResizeVmRequest]) (*connect.Response[pb.ResizeVmResponse], error) {
       // Implementation
   }
   ```

4. **Add tests**:
   ```go
   func TestResizeVm(t *testing.T) {
       // Test implementation
   }
   ```

### Adding Metrics

1. **Define metric**:
   ```go
   VmResizeTotal = prometheus.NewCounterVec(
       prometheus.CounterOpts{
           Name: "metald_vm_resize_total",
           Help: "Total number of VM resize operations",
       },
       []string{"status"},
   )
   ```

2. **Register metric**:
   ```go
   func init() {
       prometheus.MustRegister(VmResizeTotal)
   }
   ```

3. **Update metric**:
   ```go
   metrics.VmResizeTotal.WithLabelValues("success").Inc()
   ```

### Adding Configuration

1. **Update config struct**:
   ```go
   type Config struct {
       MaxVmsPerCustomer int `env:"MAX_VMS_PER_CUSTOMER" default:"100"`
   }
   ```

2. **Document in README**:
   ```markdown
   - `UNKEY_METALD_MAX_VMS_PER_CUSTOMER`: Maximum VMs per customer (default: 100)
   ```

3. **Add validation**:
   ```go
   func (c *Config) Validate() error {
       if c.MaxVmsPerCustomer < 1 {
           return errors.New("MAX_VMS_PER_CUSTOMER must be positive")
       }
       return nil
   }
   ```

---

For architecture details, see [Architecture Documentation](../architecture/)
For operations procedures, see [Operations Guide](../operations/)