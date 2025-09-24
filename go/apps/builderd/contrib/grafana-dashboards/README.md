# Builderd Grafana Dashboards

This directory contains Grafana dashboard definitions for monitoring builderd operations, performance, and health.

## Dashboard Overview

| Dashboard | Purpose | Audience |
|-----------|---------|----------|
| `builderd-overview.json` | High-level service metrics and health | Operations, SRE |
| `builderd-builds.json` | Build execution and performance metrics | Developers, Operations |
| `builderd-tenants.json` | Multi-tenant usage and quota monitoring | Account Management, Operations |
| `builderd-infrastructure.json` | System resources and infrastructure health | SRE, Infrastructure |
| `builderd-security.json` | Security events and isolation monitoring | Security, Operations |

## Quick Setup

### Import Dashboards

```bash
# Import all dashboards using Grafana API
for dashboard in *.json; do
    curl -X POST \
        -H "Authorization: Bearer $GRAFANA_API_KEY" \
        -H "Content-Type: application/json" \
        -d @"$dashboard" \
        "$GRAFANA_URL/api/dashboards/db"
done
```

### Configure Data Sources

Ensure these data sources are configured in Grafana:

1. **Prometheus** - Metrics collection
   ```yaml
   url: http://prometheus:9090
   access: proxy
   isDefault: true
   ```

2. **Loki** - Log aggregation (optional)
   ```yaml
   url: http://loki:3100
   access: proxy
   ```

3. **Jaeger** - Distributed tracing (optional)
   ```yaml
   url: http://jaeger:16686
   access: proxy
   ```

## Dashboard Details

### 1. Builderd Overview Dashboard

**File**: `builderd-overview.json`

**Key Metrics**:
- Service uptime and availability
- Build success rates and failure trends
- Active builds and queue size
- Resource utilization (CPU, Memory, Disk)
- API response times and error rates

**Panels**:
```
┌─────────────────┬─────────────────┬─────────────────┐
│ Service Status  │ Build Success   │ Active Builds   │
│ 🟢 UP 99.9%     │ Rate 97.2%      │ Queue: 3        │
│                 │                 │ Running: 12     │
├─────────────────┴─────────────────┴─────────────────┤
│ Build Duration Over Time                            │
│ ▲ ████████████████████████████████████████████████ │
├─────────────────┬─────────────────┬─────────────────┤
│ CPU Usage       │ Memory Usage    │ Disk Usage      │
│ ████████ 72%    │ ██████ 68%      │ ████ 45%       │
└─────────────────┴─────────────────┴─────────────────┘
```

### 2. Builderd Builds Dashboard

**File**: `builderd-builds.json`

**Key Metrics**:
- Build execution timeline
- Build duration distribution
- Failure analysis by source type
- Build size and optimization metrics
- Cache hit rates

**Use Cases**:
- Performance optimization
- Build failure investigation
- Capacity planning
- SLA monitoring

### 3. Builderd Tenants Dashboard

**File**: `builderd-tenants.json`

**Key Metrics**:
- Per-tenant build usage
- Quota utilization and violations
- Storage usage by tenant
- Tier-based resource consumption
- Billing-relevant metrics

**Use Cases**:
- Account management
- Quota planning
- Billing verification
- Tenant performance analysis

### 4. Builderd Infrastructure Dashboard

**File**: `builderd-infrastructure.json`

**Key Metrics**:
- Docker daemon health and performance
- Storage backend performance
- Network I/O and registry access
- Database connection pool status
- OpenTelemetry trace sampling

**Use Cases**:
- Infrastructure troubleshooting
- Capacity planning
- Performance optimization
- Dependency monitoring

### 5. Builderd Security Dashboard

**File**: `builderd-security.json`

**Key Metrics**:
- Tenant isolation violations
- Authentication and authorization events
- Resource quota violations
- Network policy violations
- Audit log analysis

**Use Cases**:
- Security monitoring
- Compliance reporting
- Incident investigation
- Policy enforcement

## Metric Reference

### Core Service Metrics

```prometheus
# Service health
up{job="builderd"}
builderd_info{version, commit}

# Build metrics
builderd_builds_total{status, tenant_id, source_type}
builderd_builds_duration_seconds{tenant_id, source_type}
builderd_builds_queue_size
builderd_builds_concurrent{tenant_id}

# Resource metrics
builderd_memory_usage_bytes
builderd_cpu_usage_percent
builderd_disk_usage_bytes{path}
builderd_disk_free_bytes{path}
```

### Tenant Metrics

```prometheus
# Quota usage
builderd_tenant_quota_usage{tenant_id, quota_type}
builderd_tenant_quota_limit{tenant_id, quota_type}
builderd_tenant_quota_violations_total{tenant_id, quota_type}

# Build metrics per tenant
builderd_tenant_builds_total{tenant_id, status}
builderd_tenant_builds_duration_seconds{tenant_id}
builderd_tenant_storage_usage_bytes{tenant_id}
```

### Docker Metrics

```prometheus
# Docker operations
builderd_docker_pulls_total{registry, tenant_id}
builderd_docker_pull_duration_seconds{registry}
builderd_docker_build_duration_seconds{tenant_id}
builderd_docker_cache_hits_total{tenant_id}
builderd_docker_cache_misses_total{tenant_id}
```

### Storage Metrics

```prometheus
# Storage operations
builderd_storage_operations_total{operation, backend}
builderd_storage_operation_duration_seconds{operation, backend}
builderd_storage_errors_total{operation, backend}

# Cache metrics
builderd_cache_size_bytes{tenant_id, cache_type}
builderd_cache_hit_ratio{tenant_id, cache_type}
builderd_cache_evictions_total{tenant_id, cache_type}
```

## Alert Integration

### Grafana Alerting

Configure alerts within dashboards using Grafana's built-in alerting:

```json
{
  "alert": {
    "name": "High Build Failure Rate",
    "frequency": "1m",
    "conditions": [
      {
        "query": {
          "queryType": "A",
          "refId": "A"
        },
        "reducer": {
          "type": "avg"
        },
        "evaluator": {
          "params": [0.1],
          "type": "gt"
        }
      }
    ],
    "message": "Build failure rate is above 10% for the last 5 minutes",
    "noDataState": "no_data",
    "executionErrorState": "alerting"
  }
}
```

### External Alert Manager

Export alerts to external systems:

```yaml
# Prometheus AlertManager rules
groups:
- name: builderd-dashboards
  rules:
  - alert: DashboardBuildFailureRate
    expr: rate(builderd_builds_total{status="failed"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
      dashboard: builderd-builds
    annotations:
      summary: "High build failure rate detected"
      grafana_url: "{{ $externalURL }}/d/builderd-builds"
```

## Customization

### Variable Templates

Add dynamic filtering using Grafana variables:

```json
{
  "templating": {
    "list": [
      {
        "name": "tenant_id",
        "type": "query",
        "query": "label_values(builderd_tenant_builds_total, tenant_id)",
        "refresh": "on_time_range_change",
        "includeAll": true,
        "allValue": ".*"
      },
      {
        "name": "time_range",
        "type": "interval",
        "query": "5m,15m,1h,6h,12h,1d",
        "auto": true,
        "auto_min": "5m"
      }
    ]
  }
}
```

### Panel Customization

Example panel configuration for build success rate:

```json
{
  "title": "Build Success Rate",
  "type": "stat",
  "targets": [
    {
      "expr": "rate(builderd_builds_total{status=\"completed\", tenant_id=~\"$tenant_id\"}[5m]) / rate(builderd_builds_total{tenant_id=~\"$tenant_id\"}[5m]) * 100",
      "legendFormat": "Success Rate"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "unit": "percent",
      "min": 0,
      "max": 100,
      "thresholds": {
        "steps": [
          {"color": "red", "value": 0},
          {"color": "yellow", "value": 90},
          {"color": "green", "value": 95}
        ]
      }
    }
  }
}
```

## Best Practices

### Dashboard Design

1. **Hierarchy**: Start with high-level overview, drill down to details
2. **Time Ranges**: Use consistent time ranges across related panels
3. **Color Coding**: Use consistent colors for status (green=good, red=bad)
4. **Units**: Always specify appropriate units for metrics
5. **Thresholds**: Set meaningful thresholds for status indicators

### Performance

1. **Query Optimization**: Use recording rules for complex queries
2. **Time Series Limits**: Limit high-cardinality labels in queries
3. **Refresh Rates**: Use appropriate refresh intervals (5-30 seconds)
4. **Panel Count**: Limit dashboards to 20-30 panels for performance

### Maintenance

1. **Version Control**: Store dashboard JSON in source control
2. **Automated Deployment**: Use infrastructure-as-code for dashboard deployment
3. **Regular Review**: Review and update dashboards quarterly
4. **User Feedback**: Collect feedback from dashboard users

## Troubleshooting

### Common Issues

#### No Data in Panels

```bash
# Check if builderd is exposing metrics
curl http://builderd:9090/metrics

# Verify Prometheus can scrape builderd
curl http://prometheus:9090/api/v1/label/__name__/values | grep builderd

# Check Grafana data source configuration
curl -H "Authorization: Bearer $GRAFANA_API_KEY" \
    http://grafana:3000/api/datasources
```

#### Dashboard Import Errors

```bash
# Validate JSON syntax
cat builderd-overview.json | jq '.'

# Check for missing data source UIDs
grep -o '"datasource":{[^}]*}' builderd-overview.json
```

#### Performance Issues

```bash
# Check query performance in Prometheus
curl -G http://prometheus:9090/api/v1/query \
    --data-urlencode 'query=rate(builderd_builds_total[5m])'

# Review Grafana query inspector for slow panels
# Access via panel menu -> Inspect -> Query
```

### Support

For dashboard issues:
1. Check the [main builderd documentation](../docs/README.md)
2. Review Grafana documentation for panel configuration
3. Verify Prometheus metrics are being exported correctly
4. Check data source connectivity and permissions

## Contributing

To contribute new dashboards or improvements:

1. Create dashboards in Grafana UI
2. Export as JSON (`Share` -> `Export` -> `Save to file`)
3. Remove data source UIDs and make generic
4. Add appropriate documentation
5. Test with clean Grafana instance
6. Submit pull request with dashboard and documentation

Dashboard naming convention: `builderd-{category}.json`

Categories: `overview`, `builds`, `tenants`, `infrastructure`, `security`, `custom-{name}`
