// Package caches provides a centralized caching service for commonly accessed database entities.
//
// The caches package is designed to improve performance by reducing database load
// for frequently accessed data. It maintains in-memory caches with configurable
// freshness and staleness windows for various database entities used throughout
// the Unkey system.
//
// Common use cases:
//   - Looking up ratelimit namespaces by name
//   - Retrieving ratelimit overrides that match specific criteria
//   - Getting API keys by their hash
//   - Retrieving permissions associated with a key ID
//
// All caches are initialized with appropriate TTL settings and size limits,
// and include OpenTelemetry tracing for observability.
package caches
