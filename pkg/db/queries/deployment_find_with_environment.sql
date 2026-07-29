-- name: FindDeploymentWithEnvironment :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT d.*, e.slug AS environment_slug
FROM deployments d
JOIN environments e ON (d.environment_id COLLATE utf8mb4_0900_ai_ci = e.id AND d.environment_id COLLATE utf8mb4_0900_as_cs = e.id)
WHERE d.id = sqlc.arg(id);
