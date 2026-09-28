-- name: FindOIDCVercelIntegrationByAudience :many
-- Loads the integration trust policy and permission grants selected by the
-- token's exact, case-sensitive audience. Permission IDs are globally unique,
-- so grants can reference the shared Unkey permission rows used by root keys.
-- The left joins preserve an integration with no grants as one row with a null slug.
SELECT ov.id,
       ov.workspace_id,
       ov.name,
       ov.audience,
       ov.vercel_team_slug,
       ov.vercel_project_name,
       ov.vercel_environment,
       ov.enabled,
       w.enabled AS workspace_enabled,
       p.slug AS permission_slug
FROM oidc_vercel ov
JOIN workspaces w
  ON w.id = ov.workspace_id
  AND w.deleted_at_m IS NULL
LEFT JOIN oidc_vercel_permissions ovp
  ON ovp.oidc_vercel_id = ov.id
  AND ovp.workspace_id = ov.workspace_id
LEFT JOIN permissions p
  ON p.id = ovp.permission_id
WHERE ov.audience = sqlc.arg(audience)
ORDER BY p.slug;
