-- name: DeleteAppById :exec
DELETE a, b
FROM apps a
LEFT JOIN app_bindings b ON b.app_id = a.id OR (b.resource_type = 'app' AND b.resource_id = a.id)
WHERE a.id = sqlc.arg(id);
