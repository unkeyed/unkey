// Package cdc consumes Vitess VStream snapshots and live row changes.
// It owns the gRPC transport, heartbeat timeout, and committed resume positions.
// Domain-specific consumers provide table/query rules and interpret FIELD and
// ROW events without losing Vitess type information or delete before-images.
//
// A Client can serve concurrent watches over one connection. Each Watch opens
// its own VStream. For example, with an initialized client and callbacks:
//
//	rules := []cdc.Rule{{Table: "records", Query: "select id, value from records"}}
//	err := client.Watch(ctx, rules, resumeToken, handleEvent, saveCheckpoint)
//
// Callbacks run in stream order. Persist a checkpoint only after preceding
// events have been applied durably. Reconnect with the last accepted token;
// an empty token starts a snapshot. Tokens contain the exact filter rules and
// full VGTID, including partial snapshot positions, and are opaque to consumers.
// They validate subscription consistency, not authorization. ErrInvalidToken
// and ErrExpired require a new snapshot rather than retrying the same token.
//
// This package does not provide a durable queue, retries, exactly-once delivery,
// or destination reconciliation. Consumers must make repeated events safe and
// reconcile stale destination records when rebuilding after a lost cursor.
package cdc
