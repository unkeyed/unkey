-- name: DeleteAppConnection :exec
DELETE c, t FROM app_connections c
LEFT JOIN connection_app_targets t ON t.connection_id = c.id
WHERE c.id = sqlc.arg(id) AND c.workspace_id = sqlc.arg(workspace_id);
