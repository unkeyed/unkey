-- name: ListDeploymentDomains :many
-- Frontline routes are the hostnames a deployment serves. Only these are
-- returned; the custom_domains classification is deliberately skipped so the
-- read stays a single index range scan on frontline_routes.deployment_id.
SELECT r.fully_qualified_domain_name AS domain
FROM frontline_routes r
WHERE r.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.fully_qualified_domain_name;
