-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    sqlc.embed(c),
    sqlc.embed(q)
FROM `clickhouse_workspace_settings` c
JOIN `quota` q ON c.workspace_id = q.workspace_id
WHERE c.workspace_id = sqlc.arg(workspace_id);
