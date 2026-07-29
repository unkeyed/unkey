-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(c),
    sqlc.embed(q)
FROM `clickhouse_workspace_settings` c
JOIN `quota` q ON (c.workspace_id COLLATE utf8mb4_0900_ai_ci = q.workspace_id AND c.workspace_id COLLATE utf8mb4_0900_as_cs = q.workspace_id)
WHERE c.workspace_id = sqlc.arg(workspace_id);
