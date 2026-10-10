-- name: DeleteAppConnectionById :execrows
DELETE b, t
FROM app_connections b
LEFT JOIN connection_app_targets t ON t.connection_id = b.id
WHERE b.id = sqlc.arg(id);
