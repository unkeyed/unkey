# Telemetry Cardinality Control Guide

## Overview

Both `billaged` and `metald` services include configurable high-cardinality controls for production safety while enabling detailed monitoring during development.

## Configuration

### Environment Variables

| Service | Variable | Default | Description |
|---------|----------|---------|-------------|
| billaged | `BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED` | `false` | Enable vm_id, customer_id labels |
| metald | `UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED` | `false` | Enable vm_id, process_id, jailer_id labels |

### Default Behavior (Production Safe)

**High-cardinality DISABLED (default):**
- ✅ Low cardinality metrics only
- ✅ Labels limited to: backend, error_type, use_jailer, exit_code
- ✅ Production safe - bounded cardinality
- ✅ Suitable for long-term retention

**High-cardinality ENABLED (development/debugging):**
- ⚠️ Includes VM IDs, process IDs, customer IDs
- ⚠️ Cardinality grows with number of VMs/customers
- ⚠️ Higher storage costs
- ⚠️ Potential performance impact on metrics systems

## Performance Implications

### Cardinality Impact

| Scenario | VMs | Customers | Est. Series (High-Cardinality) | Storage Impact |
|----------|-----|-----------|--------------------------------|----------------|
| Development | 5-10 | 1-3 | ~500-1,000 | Minimal |
| Small Production | 100 | 50 | ~50,000-100,000 | Moderate |
| Large Production | 10,000+ | 1,000+ | 10M+ | **Severe** |

### Metrics System Impact

**Prometheus:**
- High cardinality can cause memory issues
- Slow query performance
- Increased ingestion latency

**OTEL Collectors:**
- Higher CPU/memory usage
- Increased export bandwidth
- Potential backpressure

## Implementation Details

### Billaged Metrics

**High-cardinality labels (when enabled):**
```go
// Usage processing
attribute.String("vm_id", vmID)
attribute.String("customer_id", customerID)
```

**Always-present labels:**
```go
// Error classification
attribute.String("error_type", errorType)
```

### Metald Metrics

**VM Lifecycle (always low-cardinality):**
```go
// Backend and error classification only
attribute.String("backend", backend)
attribute.String("error_type", errorType)
attribute.Bool("force", force)
```

**Process Management (high-cardinality when enabled):**
```go
// When high-cardinality enabled:
attribute.String("vm_id", vmID)
attribute.String("process_id", processID)
attribute.String("jailer_id", jailerID)

// Always present:
attribute.Bool("use_jailer", useJailer)
attribute.String("error_type", errorType)
attribute.Int("exit_code", exitCode)
```

## Best Practices

### Production Deployment
1. **Keep high-cardinality DISABLED** (`false`)
2. Monitor aggregate metrics only
3. Use labels for categorization, not identification
4. Implement cardinality monitoring alerts

### Development/Debugging
1. **Enable high-cardinality temporarily** (`true`)
2. Monitor specific VM/customer behavior
3. Debug process lifecycle issues
4. Disable after investigation

### Monitoring Strategy

**Production Monitoring:**
```promql
# Service health
rate(billaged_usage_records_processed_total[5m])
rate(metald_vm_create_success_total[5m]) / rate(metald_vm_create_requests_total[5m])

# Error rates by type
rate(billaged_billing_errors_total[5m]) by (error_type)
rate(metald_vm_create_failures_total[5m]) by (backend, error_type)
```

**Development Debugging:**
```promql
# Per-VM analysis (when high-cardinality enabled)
rate(billaged_usage_records_processed_total[5m]) by (vm_id, customer_id)
rate(metald_process_create_failures_total[5m]) by (vm_id, error_type)
```

## Configuration Validation

### Error Handling

Both services now include comprehensive error logging for invalid configuration:

```bash
# Example warning output
Warning: invalid BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED 'maybe', using default false: strconv.ParseBool: parsing "maybe": invalid syntax
```

### Startup Logging

Services log their cardinality configuration at startup:

```json
{
  "level": "INFO",
  "msg": "VM and billing metrics initialized",
  "high_cardinality_enabled": false
}
```

## Migration Guide

### Enabling High-Cardinality

1. **Test in development first**
2. **Monitor metrics system resource usage**
3. **Gradually enable in staging**
4. **Consider retention policy adjustments**

### Disabling High-Cardinality

1. **Remove environment variable** (defaults to false)
2. **Restart services**
3. **Wait for old series to expire**
4. **Verify cardinality reduction**

## Troubleshooting

### High Memory Usage
- Check if high-cardinality is accidentally enabled
- Monitor series count: `prometheus_tsdb_symbol_table_size_bytes`
- Consider shorter retention for high-cardinality metrics

### Missing Detailed Metrics
- Verify high-cardinality is enabled for debugging
- Check environment variable syntax
- Review startup logs for configuration errors

### Query Performance Issues
- Reduce time range for high-cardinality queries
- Use rate() functions instead of raw counters
- Consider aggregation rules for frequent queries

## Security Considerations

- VM IDs and customer IDs may contain sensitive information
- Ensure metrics systems have appropriate access controls
- Consider metric sanitization for external exports
- Review data retention policies for compliance