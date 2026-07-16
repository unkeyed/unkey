-- name: ListDeploymentDomains :many
-- Frontline routes are the hostnames a deployment serves. The left join to
-- custom_domains classifies each as system (no match) or custom (match) and
-- surfaces the custom domain's verification status.
SELECT
  r.fully_qualified_domain_name AS domain,
  cd.verification_status AS custom_verification_status
FROM frontline_routes r
LEFT JOIN custom_domains cd
  ON cd.domain = r.fully_qualified_domain_name
  AND cd.workspace_id = sqlc.arg(workspace_id)
WHERE r.deployment_id = sqlc.arg(deployment_id);
