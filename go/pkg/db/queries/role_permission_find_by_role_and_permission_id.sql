-- name: FindRolePermissionByRoleAndPermissionID :many
SELECT *
FROM roles_permissions
WHERE role_id = sqlc.arg(role_id)
  AND permission_id = sqlc.arg(permission_id);