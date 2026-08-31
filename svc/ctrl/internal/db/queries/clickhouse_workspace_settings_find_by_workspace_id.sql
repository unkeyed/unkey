-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    c.workspace_id AS clickhouse_workspace_id,
    c.username AS clickhouse_username,
    c.password_encrypted AS clickhouse_password_encrypted,
    c.quota_duration_seconds AS clickhouse_quota_duration_seconds,
    c.max_queries_per_window AS clickhouse_max_queries_per_window,
    c.max_execution_time_per_window AS clickhouse_max_execution_time_per_window,
    c.max_query_execution_time AS clickhouse_max_query_execution_time,
    c.max_query_memory_bytes AS clickhouse_max_query_memory_bytes,
    c.max_query_result_rows AS clickhouse_max_query_result_rows,
    l.logs_retention_days_max AS quota_logs_retention_days
FROM `clickhouse_workspace_settings` c
JOIN `limits` l ON l.workspace_id = c.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
