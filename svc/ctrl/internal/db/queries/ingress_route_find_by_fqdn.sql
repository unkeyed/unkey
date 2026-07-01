-- name: FindFrontlineRouteByFQDN :one
SELECT * FROM frontline_routes WHERE fully_qualified_domain_name = ?;
