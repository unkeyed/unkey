// Package collector queries the Kubernetes Metrics Server for per-pod CPU/memory
// snapshots and writes them to ClickHouse via the shared [clickhouse.Bufferer].
// Pod labels and resource limits come from the informer cache.
package collector
