-- name: DeleteAppById :exec
DELETE a, b, t
FROM apps a
LEFT JOIN app_connections b ON b.app_id = a.id OR (b.resource_type = 'app' AND b.resource_id = a.id)
LEFT JOIN connection_app_targets t ON t.connection_id = b.id
WHERE a.id = sqlc.arg(id);
