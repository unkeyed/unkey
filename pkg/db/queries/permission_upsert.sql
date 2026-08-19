-- name: UpsertPermission :exec
-- Inserts a permission or leaves the existing workspace/slug row unchanged.
-- Use FindPermissionsBySlugsForUpdate after this to get the canonical row.
INSERT INTO permissions (
  id,
  workspace_id,
  project_id,
  name,
  slug,
  description,
  created_at_m
)
VALUES (
  sqlc.arg(permission_id),
  sqlc.arg(workspace_id),
  sqlc.arg(project_id),
  sqlc.arg(name),
  sqlc.arg(slug),
  sqlc.arg(description),
  sqlc.arg(created_at_m)
)
ON DUPLICATE KEY UPDATE slug = slug;
