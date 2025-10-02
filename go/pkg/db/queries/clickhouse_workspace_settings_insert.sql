-- name: InsertClickhouseWorkspaceSettings :exec
INSERT INTO `clickhouse_workspace_settings` (
    workspace_id,
    username,
    password_encrypted,
    quota_duration_seconds,
    max_queries_per_window,
    max_execution_time_per_window,
    max_query_execution_time,
    max_query_memory_bytes,
    max_query_result_rows,
    max_rows_to_read,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(username),
    sqlc.arg(password_encrypted),
    sqlc.arg(quota_duration_seconds),
    sqlc.arg(max_queries_per_window),
    sqlc.arg(max_execution_time_per_window),
    sqlc.arg(max_query_execution_time),
    sqlc.arg(max_query_memory_bytes),
    sqlc.arg(max_query_result_rows),
    sqlc.arg(max_rows_to_read),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
