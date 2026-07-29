-- name: FindDeploymentWithEnvironmentAndApp :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT d.*, e.slug AS environment_slug, a.current_deployment_id, a.is_rolled_back
FROM deployments d
JOIN environments e ON (e.id = d.environment_id COLLATE utf8mb4_0900_ai_ci AND e.id = d.environment_id COLLATE utf8mb4_0900_as_cs)
JOIN apps a ON (a.id = d.app_id COLLATE utf8mb4_0900_ai_ci AND a.id = d.app_id COLLATE utf8mb4_0900_as_cs)
WHERE d.id = sqlc.arg(id);
