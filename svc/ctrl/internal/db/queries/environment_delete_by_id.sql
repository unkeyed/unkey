-- name: DeleteEnvironmentById :exec
DELETE e, b, bt
FROM environments e
LEFT JOIN app_connections b ON b.environment_id = e.id
LEFT JOIN connection_app_targets bt ON bt.connection_id = b.id
WHERE e.id = sqlc.arg(id);
