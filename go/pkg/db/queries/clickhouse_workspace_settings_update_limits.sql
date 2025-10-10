-- name: UpdateClickhouseWorkspaceSettingsLimits :exec
UPDATE `clickhouse_workspace_settings`
SET
    quota_duration_seconds = sqlc.arg(quota_duration_seconds),
    max_queries_per_window = sqlc.arg(max_queries_per_window),
    max_execution_time_per_window = sqlc.arg(max_execution_time_per_window),
    max_query_execution_time = sqlc.arg(max_query_execution_time),
    max_query_memory_bytes = sqlc.arg(max_query_memory_bytes),
    max_query_result_rows = sqlc.arg(max_query_result_rows),
    max_rows_to_read = sqlc.arg(max_rows_to_read),
    updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id);
