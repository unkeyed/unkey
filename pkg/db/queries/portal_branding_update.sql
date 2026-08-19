-- name: UpdatePortalBranding :exec
-- Branding lives on the portal row, so the dashboard's branding form is one
-- write against an existing portal rather than an upsert into a side table.
-- Discrete columns rather than a JSON blob, so each field is typed and length
-- bounded by the database.
UPDATE portals
SET logo_url = sqlc.narg(logo_url),
    primary_color = sqlc.narg(primary_color),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);
