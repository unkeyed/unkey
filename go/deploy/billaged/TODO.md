# Billaged TODO

## High Priority

- [ ] Implement proper ClickHouse schema migrations
  - Version tracking for schema changes
  - Rollback capabilities
  - Migration testing framework

- [ ] Add rate limiting for billing events
  - Per-tenant rate limits
  - Circuit breaker for ClickHouse writes
  - Backpressure handling

## Medium Priority

- [ ] Implement billing event deduplication
  - Idempotency keys for events
  - Duplicate detection window
  - Metrics for duplicate events

- [ ] Add billing aggregation optimizations
  - Pre-aggregated materialized views
  - Configurable aggregation windows
  - Real-time vs batch aggregation modes

- [ ] Implement data retention policies
  - Configurable retention per event type
  - Automated data archival
  - Compliance with data regulations

## Low Priority

- [ ] Add support for multiple ClickHouse clusters
  - Read/write splitting
  - Cluster health monitoring
  - Automatic failover

- [ ] Implement billing event replay
  - Event sourcing capabilities
  - Point-in-time recovery
  - Audit trail for billing changes

- [ ] Add billing analytics endpoints
  - Cost breakdown by resource
  - Usage trends and forecasting
  - Anomaly detection

## Completed

- [x] Basic service implementation
- [x] ClickHouse integration
- [x] ConnectRPC API
- [x] Event aggregation
- [x] Prometheus metrics
- [x] SPIFFE/mTLS support
- [x] Grafana dashboards
- [x] Unified health endpoint
- [x] Unified Makefile structure