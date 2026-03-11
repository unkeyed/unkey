-- name: FindFrontlineRouteByFQDN :one
-- FindFrontlineRouteByFQDN resolves an incoming hostname to the routing target
-- used by frontline. The query intentionally projects only environment and
-- deployment identifiers because callers do not need the full route record,
-- and a narrow projection reduces row decoding work on the request path.
--
-- The hostname column is unique in schema, so this query returns at most one
-- row for a given input hostname.
--
-- Example: with fully_qualified_domain_name 'api.example.com', the query
-- returns the single environment_id and deployment_id currently bound to that
-- domain.
SELECT
  environment_id,
  deployment_id
FROM frontline_routes
WHERE fully_qualified_domain_name = ?;
