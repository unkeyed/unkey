-- name: CountInstancesByAppId :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT COUNT(*) as count
FROM instances i
JOIN deployments d ON (i.deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND i.deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
WHERE d.app_id = sqlc.arg(app_id);
