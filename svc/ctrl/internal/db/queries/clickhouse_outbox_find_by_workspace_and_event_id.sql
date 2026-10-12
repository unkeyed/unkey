-- name: FindClickhouseOutboxByWorkspaceAndEventId :one
-- FindClickhouseOutboxByWorkspaceAndEventId returns one outbox row, including
-- rows the drainer already marked deleted. Only tests use this query.
SELECT pk, version, workspace_id, event_id, payload, created_at, deleted_at
FROM `clickhouse_outbox`
WHERE workspace_id = sqlc.arg(workspace_id) AND event_id = sqlc.arg(event_id);
