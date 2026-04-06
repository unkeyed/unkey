// Package collector reads cgroup v2 files for per-pod CPU/memory usage and
// writes resource snapshots to ClickHouse. Network egress comes from conntrack.
// Pod labels and resource limits come from the informer cache.
package collector
