-- name: DeleteProjectById :exec
DELETE FROM projects WHERE id = sqlc.arg(id) AND slug <> 'default';
