-- name: InsertClickhouseOutboxWithResultTarget :exec
-- transactional-batch-statement
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
    JSON_SET(CAST(sqlc.arg(payload) AS JSON), '$.targets[1].id', sqlc.arg(result_target_id)),
    sqlc.arg(created_at)
);
