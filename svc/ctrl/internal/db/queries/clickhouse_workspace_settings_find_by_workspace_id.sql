-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(c),
    sqlc.embed(q)
FROM `clickhouse_workspace_settings` c
JOIN `quota` q ON (q.workspace_id = c.workspace_id COLLATE utf8mb4_0900_ai_ci AND q.workspace_id = c.workspace_id COLLATE utf8mb4_0900_as_cs)
WHERE c.workspace_id = sqlc.arg(workspace_id);
