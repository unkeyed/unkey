-- name: InsertRolePermission :exec
INSERT INTO roles_permissions (
  role_id,
  permission_id
)
VALUES (
  sqlc.arg(role_id),
  sqlc.arg(permission_id)
)
;
