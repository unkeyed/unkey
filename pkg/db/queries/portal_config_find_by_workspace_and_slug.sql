-- name: FindPortalConfigByWorkspaceAndSlug :one
SELECT portal_configurations.pk, portal_configurations.id, portal_configurations.workspace_id, portal_configurations.slug, portal_configurations.app_id, portal_configurations.key_auth_id, portal_configurations.enabled, portal_configurations.return_url, portal_configurations.created_at, portal_configurations.updated_at FROM portal_configurations
WHERE workspace_id = sqlc.arg(workspace_id) AND slug = sqlc.arg(slug);
