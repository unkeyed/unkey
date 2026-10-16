-- name: DeletePortalsByAppID :exec
DELETE p, s, o
FROM portals p
LEFT JOIN portal_sessions s ON s.portal_id = p.id
LEFT JOIN openapi_specs o ON o.portal_id = p.id
WHERE p.app_id = sqlc.arg(app_id);

-- name: DeletePortalsByProjectID :exec
DELETE p, s, o
FROM portals p
LEFT JOIN portal_sessions s ON s.portal_id = p.id
LEFT JOIN openapi_specs o ON o.portal_id = p.id
WHERE p.project_id = sqlc.arg(project_id);
