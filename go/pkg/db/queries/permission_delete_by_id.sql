-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = sqlc.arg(permission_id);
