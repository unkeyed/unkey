-- name: ListClickhouseWorkspaceIDs :many
SELECT workspace_id
FROM clickhouse_workspace_settings
ORDER BY workspace_id;
