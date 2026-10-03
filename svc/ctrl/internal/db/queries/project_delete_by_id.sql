-- name: DeleteProjectById :exec
DELETE p, b
FROM projects p
LEFT JOIN app_bindings b ON b.project_id = p.id
WHERE p.id = sqlc.arg(id);
