# Billaged Grafana Dashboards

This directory contains Grafana dashboards for monitoring the billaged service.

## Available Dashboards

### billaged-overview.json
A comprehensive overview dashboard for the billaged billing aggregation service that provides:

#### Key Metrics
- **Usage Records Rate**: Rate of usage records being processed per second
- **Avg Aggregation Duration**: Average time spent aggregating usage metrics
- **Active VMs**: Current number of VMs being tracked for billing
- **Error Rate**: Rate of billing processing errors

#### Detailed Views
- **Usage Processing**: Time-series charts showing usage record processing rates and aggregation duration percentiles
- **Active VMs & Errors**: VM count tracking and error breakdown by type
- **System Health**: CPU usage, memory usage, and goroutine counts

## Importing Dashboards

1. **Manual Import**:
   ```bash
   # Open Grafana UI
   # Go to Dashboards -> Import
   # Upload the JSON file or paste its contents
   ```

2. **Automated Import** (if you have Grafana API access):
   ```bash
   # Using curl to import via API
   curl -X POST \
     http://your-grafana-instance:3000/api/dashboards/db \
     -H 'Content-Type: application/json' \
     -H 'Authorization: Bearer YOUR_API_KEY' \
     -d @billaged-overview.json
   ```

## Metrics Used

The dashboards expect the following Prometheus metrics from billaged:

### Core Billing Metrics
- `billaged_usage_records_processed_total` - Counter of processed usage records
- `billaged_aggregation_duration_seconds` - Histogram of aggregation durations
- `billaged_active_vms` - Gauge of active VMs being tracked
- `billaged_billing_errors_total` - Counter of billing errors by type

### System Health Metrics (from Prometheus Go client)
- `process_cpu_seconds_total` - Process CPU usage
- `process_resident_memory_bytes` - Process memory usage
- `go_goroutines` - Number of active goroutines

## Configuration

### High-Cardinality Labels
The dashboards are designed to work with both high and low cardinality configurations:

- **High-cardinality enabled**: Shows per-VM breakdowns when `BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=true`
- **High-cardinality disabled**: Shows aggregate metrics only when `BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false` (default)

### Rate Intervals
All rate calculations use `$__rate_interval` for optimal performance across different time ranges.

## Troubleshooting

1. **No Data Showing**: 
   - Verify billaged is running with `BILLAGED_OTEL_ENABLED=true`
   - Check Prometheus is scraping billaged metrics endpoint
   - Confirm the job name in Prometheus matches "billaged"

2. **Missing Metrics**:
   - Ensure billaged is processing usage records (metrics only appear after first usage)
   - Check billaged logs for initialization errors

3. **Performance Issues**:
   - Consider disabling high-cardinality labels in production
   - Adjust dashboard refresh rates for heavy load scenarios