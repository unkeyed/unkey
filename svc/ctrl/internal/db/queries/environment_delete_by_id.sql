-- name: DeleteEnvironmentById :exec
DELETE e, b
FROM environments e
LEFT JOIN app_bindings b ON b.environment_id = e.id
WHERE e.id = sqlc.arg(id);
