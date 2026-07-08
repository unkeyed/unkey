-- name: DeleteFrontlineRouteByFQDNAndProject :exec
DELETE FROM frontline_routes
WHERE fully_qualified_domain_name = sqlc.arg(fqdn)
  AND project_id = sqlc.arg(project_id);
