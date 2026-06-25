-- name: DeleteRoleByID :exec
DELETE FROM roles
WHERE id = sqlc.arg(role_id);