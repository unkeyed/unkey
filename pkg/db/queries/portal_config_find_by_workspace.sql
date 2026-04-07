-- name: FindPortalConfigByWorkspaceID :one
SELECT * FROM portal_configurations WHERE workspace_id = ? LIMIT 1;
