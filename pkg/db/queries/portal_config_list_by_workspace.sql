-- name: ListPortalConfigsByWorkspace :many
-- Lists every portal configuration in a workspace, left-joining its 1:1
-- branding row. Branding columns are null when a config has no branding. The
-- WHERE clause scopes the listing to the caller's workspace.
SELECT
    sqlc.embed(pc),
    b.logo_url AS logo_url,
    b.primary_color AS primary_color
FROM portal_configurations pc
LEFT JOIN portal_branding b ON b.portal_config_id = pc.id
WHERE pc.workspace_id = sqlc.arg(workspace_id)
ORDER BY pc.created_at DESC;
