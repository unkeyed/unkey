# AssetManagerd TODO

## High Priority

- [ ] Add Grafana dashboards for monitoring
  - Asset registration/deletion rates
  - Storage usage by type
  - Garbage collection metrics
  - API latency percentiles

- [ ] Add packaging infrastructure
  - Create debian/ directory with control files
  - Create RPM spec file
  - Add Makefile targets for package building

## Medium Priority

- [ ] Implement S3 backend storage
  - Already designed in storage interface
  - Add AWS SDK dependencies
  - Configuration for bucket/prefix

- [ ] Add asset replication
  - Cross-region replication for availability
  - Configurable replication factor
  - Health checks for replicas

- [ ] Implement content deduplication
  - Use SHA256 for content addressing
  - Reference counting for deduplicated assets
  - Migration tool for existing assets

## Low Priority

- [ ] Add remote asset sources
  - HTTP/HTTPS download support
  - S3 download support
  - Caching and retry logic

- [ ] Implement asset versioning
  - Version history tracking
  - Rollback capabilities
  - Version garbage collection

- [ ] Add asset compression
  - Transparent compression/decompression
  - Multiple compression algorithm support
  - Storage savings metrics

## Completed

- [x] Basic service implementation
- [x] Local storage backend
- [x] SQLite database for metadata
- [x] Garbage collection
- [x] ConnectRPC API
- [x] Prometheus metrics
- [x] SPIFFE/mTLS support
- [x] Integration with metald
- [x] Unified health endpoint