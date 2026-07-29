-- name: FindLiveApiByID :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT apis.*, sqlc.embed(ka)
FROM apis
JOIN key_auth as ka ON (ka.id = apis.key_auth_id COLLATE utf8mb4_0900_ai_ci AND ka.id = apis.key_auth_id COLLATE utf8mb4_0900_as_cs)
WHERE apis.id = ?
    AND ka.deleted_at_m IS NULL
    AND apis.deleted_at_m IS NULL
LIMIT 1;


