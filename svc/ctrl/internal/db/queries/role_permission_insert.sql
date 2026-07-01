-- name: InsertRolePermission :exec
INSERT INTO roles_permissions (
  role_id,
  permission_id,
  workspace_id,
  created_at_m
)
VALUES (
  sqlc.arg(role_id),
  sqlc.arg(permission_id),
  sqlc.arg(workspace_id),
  sqlc.arg(created_at_m)
)
;
