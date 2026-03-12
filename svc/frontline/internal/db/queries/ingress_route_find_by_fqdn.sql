-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves a hostname to the environment and
-- deployment IDs frontline needs to route a request.
-- The projection intentionally stays narrow because this lookup is on the
-- request path and callers do not need the rest of the route row.
SELECT
  environment_id,
  deployment_id
FROM frontline_routes
WHERE fully_qualified_domain_name = ?;
