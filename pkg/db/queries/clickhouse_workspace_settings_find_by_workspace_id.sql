-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    sqlc.embed(c),
    sqlc.embed(l)
FROM `clickhouse_workspace_settings` c
JOIN `limits` l ON c.workspace_id = l.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
