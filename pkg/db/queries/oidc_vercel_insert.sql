-- name: InsertOIDCVercelIntegration :exec
-- InsertOIDCVercelIntegration binds a customer workspace to an exact audience
-- and Vercel trust policy. The audience and workspace trust tuple must be unique.
INSERT INTO oidc_vercel (
    id, workspace_id, name, audience, vercel_team_slug,
    vercel_project_name, vercel_environment, enabled, created_at
) VALUES (
    sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(audience),
    sqlc.arg(vercel_team_slug), sqlc.arg(vercel_project_name),
    sqlc.arg(vercel_environment), sqlc.arg(enabled), sqlc.arg(created_at)
);
