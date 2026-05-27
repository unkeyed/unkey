-- name: ListClickhouseOutboxByWorkspace :many
-- ListClickhouseOutboxByWorkspace returns every outbox row queued for a
-- workspace, regardless of drainer state. Intended for tests and ad-hoc
-- inspection (the live drainer uses FindClickhouseOutboxBatch which locks
-- and filters by version).
SELECT pk, version, workspace_id, event_id, payload, created_at
FROM clickhouse_outbox
WHERE workspace_id = sqlc.arg(workspace_id)
ORDER BY pk;
