-- name: InsertClickhouseOutbox :exec
-- InsertClickhouseOutbox enqueues one event for ClickHouse export. Called
-- from the same MySQL transaction as the underlying mutation, so durability
-- is exactly the durability of the mutation: if the mutation commits, the
-- outbox row commits.
--
-- version namespaces the payload schema (e.g. "audit_log.v1"). The drainer
-- filters by versions it knows, so writing a new version without a matching
-- drainer leaves rows queued safely.
INSERT INTO `clickhouse_outbox` (
    version,
    workspace_id,
    event_id,
    payload,
    created_at
) VALUES (
    sqlc.arg(version),
    sqlc.arg(workspace_id),
    sqlc.arg(event_id),
    CAST(sqlc.arg(payload) AS JSON),
    sqlc.arg(created_at)
);
