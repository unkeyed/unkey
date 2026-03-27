-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves a hostname to the routing information
-- frontline needs to forward a request. For deployment routes this includes
-- environment and deployment IDs; for portal routes it includes the portal
-- config ID and path prefix. The route_type discriminator tells the caller
-- which set of nullable columns to use.
SELECT
  route_type,
  environment_id,
  deployment_id,
  portal_config_id,
  path_prefix
FROM frontline_routes
WHERE fully_qualified_domain_name = ?;
