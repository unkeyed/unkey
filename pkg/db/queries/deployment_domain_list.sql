-- name: ListDeploymentDomains :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT r.fully_qualified_domain_name AS domain
FROM frontline_routes r
JOIN deployments d ON (r.deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND r.deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND r.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.fully_qualified_domain_name;

-- name: ListDeploymentDomainsByIds :many
SELECT r.deployment_id AS deployment_id, r.fully_qualified_domain_name AS domain
FROM frontline_routes r
JOIN deployments d ON (r.deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND r.deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND r.deployment_id IN (sqlc.slice('deployment_ids'))
ORDER BY r.deployment_id, r.fully_qualified_domain_name;
