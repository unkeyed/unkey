-- name: DeleteAppById :exec
DELETE FROM apps WHERE id = sqlc.arg(id);
