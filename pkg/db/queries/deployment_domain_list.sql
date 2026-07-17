-- name: ListDeploymentDomains :many
SELECT r.fully_qualified_domain_name AS domain
FROM frontline_routes r
WHERE r.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.fully_qualified_domain_name;

-- name: ListDeploymentDomainsByIds :many
SELECT r.deployment_id AS deployment_id, r.fully_qualified_domain_name AS domain
FROM frontline_routes r
WHERE r.deployment_id IN (sqlc.slice('deployment_ids'))
ORDER BY r.deployment_id, r.fully_qualified_domain_name;
