-- name: UpdatePortalBranding :exec
-- Branding lives on the portal row, so the dashboard's branding form is one
-- write against an existing portal rather than an upsert into a side table.
UPDATE portals
SET branding = sqlc.narg(branding),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);
