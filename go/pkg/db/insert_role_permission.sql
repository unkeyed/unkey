-- name: InsertRolePermission :one
-- Inserts a new role-permission relationship
-- Returns: The newly created role-permission record
INSERT INTO "role_permissions" (
  "roleId",
  "permissionId"
)
VALUES (
  $1, -- roleId
  $2  -- permissionId
)
RETURNING *;