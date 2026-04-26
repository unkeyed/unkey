-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves a hostname to the environment and
-- deployment IDs frontline needs to route a request, plus the edge-redirect
-- config blob attached to the route. The blob is parsed once at cache fill
-- time, never on the hot path.
SELECT
  environment_id,
  deployment_id,
  edge_redirect_config
FROM frontline_routes
WHERE fully_qualified_domain_name = ?;
