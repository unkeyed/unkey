-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    c.workspace_id AS clickhouse_workspace_id,
    c.username AS clickhouse_username,
    c.password_encrypted AS clickhouse_password_encrypted,
    c.max_query_result_rows AS clickhouse_max_query_result_rows,
    l.logs_retention_days_max AS quota_logs_retention_days
FROM `clickhouse_workspace_settings` c
JOIN `limits` l ON c.workspace_id = l.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
