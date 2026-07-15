-- name: FindPortalConfigByWorkspaceAndSlug :one
SELECT * FROM portal_configurations
WHERE workspace_id = sqlc.arg(workspace_id) AND slug = sqlc.arg(slug);
