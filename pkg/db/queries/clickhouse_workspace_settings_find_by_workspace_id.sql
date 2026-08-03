-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    c.workspace_id AS clickhouse_workspace_id,
    c.username AS clickhouse_username,
    c.password_encrypted AS clickhouse_password_encrypted,
    c.max_query_result_rows AS clickhouse_max_query_result_rows,
    q.logs_retention_days AS quota_logs_retention_days
FROM `clickhouse_workspace_settings` c
JOIN `quota` q ON c.workspace_id = q.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
