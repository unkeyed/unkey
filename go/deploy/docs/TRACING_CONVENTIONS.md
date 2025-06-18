# Tracing Conventions for Unkey Services

This document defines the unified tracing conventions for all Unkey Go services.

## Tracer Naming

Each service should create its tracer with the service name:

```go
tracer := otel.Tracer("<service-name>")
```

Examples:
- `otel.Tracer("metald")`
- `otel.Tracer("billaged")`
- `otel.Tracer("builderd")`
- `otel.Tracer("assetmanagerd")`

## Span Naming Conventions

### RPC Server Spans

For ConnectRPC server spans, use a hierarchical naming format:

```
<service>.<method>
```

Where:
- `<service>` is the service name (e.g., `metald`, `billaged`)
- `<method>` is the RPC method name without the full path

Examples:
- `metald.CreateVm` (not `/vmprovisioner.v1.VmService/CreateVm`)
- `billaged.RecordMetric` (not `/billing.v1.BillingService/RecordMetric`)
- `assetmanagerd.ListAssets` (not `/asset.v1.AssetManagerService/ListAssets`)

### Internal Operation Spans

For internal operations, use descriptive names with the operation context:

```
<service>.<component>.<operation>
```

Examples:
- `metald.process.create`
- `metald.network.setup`
- `billaged.metrics.flush`
- `builderd.docker.extract_filesystem`

### Database Operation Spans

```
<service>.db.<operation>
```

Examples:
- `metald.db.get_vm`
- `metald.db.update_vm_state`
- `assetmanagerd.db.register_asset`

## Span Attributes

### Required Attributes for RPC Spans

```go
trace.WithAttributes(
    attribute.String("rpc.system", "connect_rpc"),
    attribute.String("rpc.service", "<full-service-name>"),  // e.g., "vmprovisioner.v1.VmService"
    attribute.String("rpc.method", "<method-name>"),         // e.g., "CreateVm"
)
```

### Required Attributes for Internal Spans

```go
trace.WithAttributes(
    attribute.String("service.name", "<service>"),
    attribute.String("operation.type", "<type>"),  // e.g., "process", "network", "db"
    // Add operation-specific attributes
)
```

## Implementation Example

```go
// In interceptor.go
func NewOTELInterceptor(serviceName string) connect.UnaryInterceptorFunc {
    tracer := otel.Tracer(serviceName)
    
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
            // Extract method name from procedure
            procedure := req.Spec().Procedure
            methodName := extractMethodName(procedure)  // e.g., "CreateVm" from "/vmprovisioner.v1.VmService/CreateVm"
            
            // Create span with unified naming
            spanName := fmt.Sprintf("%s.%s", serviceName, methodName)
            ctx, span := tracer.Start(ctx, spanName,
                trace.WithSpanKind(trace.SpanKindServer),
                trace.WithAttributes(
                    attribute.String("rpc.system", "connect_rpc"),
                    attribute.String("rpc.service", extractServiceName(procedure)),
                    attribute.String("rpc.method", methodName),
                ),
            )
            defer span.End()
            
            // ... rest of implementation
        }
    }
}

// Helper to extract method name from full procedure path
func extractMethodName(procedure string) string {
    parts := strings.Split(procedure, "/")
    if len(parts) > 0 {
        return parts[len(parts)-1]
    }
    return procedure
}

// Helper to extract service name from full procedure path
func extractServiceName(procedure string) string {
    // "/vmprovisioner.v1.VmService/CreateVm" -> "vmprovisioner.v1.VmService"
    parts := strings.Split(procedure, "/")
    if len(parts) >= 2 {
        return parts[1]
    }
    return ""
}
```

## Migration Checklist

- [ ] Update tracer initialization in each service
- [ ] Update RPC interceptors to use unified span naming
- [ ] Update internal operation spans to follow conventions
- [ ] Add proper span attributes
- [ ] Test with Grafana Tempo to ensure traces are properly linked

## AIDEV-NOTE: Implementation Priority

1. Fix metald's outdated tracer name ("cloud-hypervisor-controlplane/rpc" -> "metald")
2. Update all RPC interceptors to use service-prefixed span names
3. Ensure consistent attribute naming across all services