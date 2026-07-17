-- name: ListDeploymentDomains :many
SELECT r.fully_qualified_domain_name AS domain
FROM frontline_routes r
WHERE r.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.fully_qualified_domain_name;
