-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves a hostname to the routing data frontline
-- needs on the request path: the deployment ID, the policy bytes the engine
-- evaluates, the upstream protocol used to pick a transport, the deployment's
-- desired state (a stopped/archived deployment returns a distinct offline 503
-- rather than the transient no_running_instances), and the workspace's
-- spend-cap suspension flag so a deployment paused for hitting the spend limit
-- returns a billing 402 instead of a generic offline. Joining deployments and
-- the workspace's billing row here keeps the fast path to a single round trip.
SELECT
  fr.environment_id,
  fr.deployment_id,
  d.sentinel_config,
  d.upstream_protocol,
  d.desired_state,
  wb.spend_suspended
FROM frontline_routes fr
INNER JOIN deployments d ON fr.deployment_id = d.id
LEFT JOIN workspace_billing wb ON wb.workspace_id = d.workspace_id
WHERE fr.fully_qualified_domain_name = sqlc.arg(fqdn);
