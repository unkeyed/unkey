-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT * FROM `clickhouse_workspace_settings`
WHERE workspace_id = sqlc.arg(workspace_id);
