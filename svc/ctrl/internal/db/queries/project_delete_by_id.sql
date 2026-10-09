-- name: DeleteProjectById :exec
DELETE p, b, t
FROM projects p
LEFT JOIN app_connections b ON b.project_id = p.id
LEFT JOIN connection_app_targets t ON t.connection_id = b.id
WHERE p.id = sqlc.arg(id);
