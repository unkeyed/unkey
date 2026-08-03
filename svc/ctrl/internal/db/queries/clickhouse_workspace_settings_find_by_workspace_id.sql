-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    c.workspace_id AS clickhouse_workspace_id,
    c.username AS clickhouse_username,
    c.password_encrypted AS clickhouse_password_encrypted,
    q.logs_retention_days AS quota_logs_retention_days
FROM `clickhouse_workspace_settings` c
JOIN `quota` q ON q.workspace_id = c.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
