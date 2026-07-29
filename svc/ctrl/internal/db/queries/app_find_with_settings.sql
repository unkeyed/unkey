-- name: FindAppWithRuntimeSettings :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(a),
    sqlc.embed(ars)
FROM apps a
INNER JOIN app_runtime_settings ars ON (ars.app_id = a.id COLLATE utf8mb4_0900_ai_ci AND ars.app_id = a.id COLLATE utf8mb4_0900_as_cs) AND ars.environment_id = sqlc.arg(environment_id)
WHERE a.id = sqlc.arg(id);
