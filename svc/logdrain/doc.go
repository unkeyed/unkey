// Package logdrain forwards customer logs out of ClickHouse to third-party
// observability providers.
//
// The service is the batch arm of the log-drain feature. It tails the
// runtime_logs_raw_v1 and sentinel_requests_raw_v1 tables on a polling
// cycle, applies per-drain filters, transforms records into the provider's
// wire format, and pushes them with retry and backoff. Drain configuration,
// credentials, state, and per-group cursors live in MySQL (log_drains,
// log_drain_credentials, log_drain_state, log_drain_cursors). Pasted tokens
// and OAuth refresh tokens are encrypted via svc/vault and only ever
// decrypted in memory inside this process.
//
// Read amplification is bounded by ClickHouse query count per (workspace,
// project, environment, source) group rather than per drain: 100 drains
// attached to the same group share one query and fan out in memory.
//
// v1 ships as a single replica in us-east-1 (co-located with ClickHouse
// Cloud). Horizontal scale is a StatefulSet with N pods; each pod's
// ordinal carves out a contiguous range of the fixed
// coordinator.TotalShards (= 64) workspace partition, so the partition
// itself never changes at runtime — only the range boundaries each pod
// owns.
package logdrain
