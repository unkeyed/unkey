-- name: InsertPermission :exec
INSERT INTO permissions (
  id,
  workspace_id,
  name,
  slug,
  description,
  created_at_m
)
VALUES (
  sqlc.arg(permission_id),
  sqlc.arg(workspace_id),
  sqlc.arg(name),
  sqlc.arg(slug),
  sqlc.arg(description),
  sqlc.arg(created_at_m)
)
;
