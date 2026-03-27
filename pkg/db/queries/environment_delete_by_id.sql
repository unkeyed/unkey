-- name: DeleteEnvironmentById :exec
DELETE FROM environments WHERE id = sqlc.arg(id);
