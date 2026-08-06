-- name: FindPortalByWorkspaceAndIdOrSlug :one
SELECT * FROM portals
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg(portal) OR slug = sqlc.arg(portal))
LIMIT 1;
