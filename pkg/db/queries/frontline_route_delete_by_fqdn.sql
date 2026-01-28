-- name: DeleteFrontlineRouteByFQDN :exec
DELETE FROM frontline_routes WHERE fully_qualified_domain_name = sqlc.arg(fqdn);
