-- name: FindRolePermissionByRoleAndPermissionID :many
SELECT roles_permissions.pk, roles_permissions.role_id, roles_permissions.permission_id, roles_permissions.workspace_id, roles_permissions.created_at_m, roles_permissions.updated_at_m
FROM roles_permissions
WHERE role_id = sqlc.arg(role_id)
  AND permission_id = sqlc.arg(permission_id);
