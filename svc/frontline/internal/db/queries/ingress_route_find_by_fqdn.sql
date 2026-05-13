-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves a hostname to the routing data frontline
-- needs on the request path: the deployment ID, the policy bytes the engine
-- evaluates, and the upstream protocol used to pick a transport. Joining
-- deployments here keeps the fast path to a single round trip.
SELECT
  fr.environment_id,
  fr.deployment_id,
  d.sentinel_config,
  d.upstream_protocol
FROM frontline_routes fr
INNER JOIN deployments d ON fr.deployment_id = d.id
WHERE fr.fully_qualified_domain_name = sqlc.arg(fqdn);
