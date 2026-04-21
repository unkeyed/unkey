-- name: FindFrontlineRouteByPortalConfigID :one
SELECT * FROM frontline_routes WHERE portal_config_id = ? AND route_type = 'portal' LIMIT 1;
