-- name: FindKeyAuthsByIds :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT ka.id as key_auth_id, a.id as api_id
FROM apis a
JOIN key_auth as ka ON (ka.id = a.key_auth_id COLLATE utf8mb4_0900_ai_ci AND ka.id = a.key_auth_id COLLATE utf8mb4_0900_as_cs)
WHERE a.workspace_id = sqlc.arg(workspace_id)
    AND a.id IN (sqlc.slice(api_ids))
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL;
