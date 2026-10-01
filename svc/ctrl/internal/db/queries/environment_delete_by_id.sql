-- name: DeleteEnvironmentById :exec
DELETE e, b
FROM environments e
LEFT JOIN app_connections b ON b.environment_id = e.id
WHERE e.id = sqlc.arg(id);
