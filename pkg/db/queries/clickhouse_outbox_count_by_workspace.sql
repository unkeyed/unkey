-- name: CountClickhouseOutboxByWorkspace :one
-- CountClickhouseOutboxByWorkspace counts every clickhouse_outbox row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM clickhouse_outbox
WHERE workspace_id = sqlc.arg(workspace_id);
