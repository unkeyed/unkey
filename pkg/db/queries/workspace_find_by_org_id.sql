-- name: FindWorkspaceByOrgID :one
SELECT workspaces.pk, workspaces.id, workspaces.org_id, workspaces.name, workspaces.slug, workspaces.k8s_namespace, workspaces.beta_features, workspaces.subscriptions, workspaces.enabled, workspaces.delete_protection, workspaces.created_at_m, workspaces.updated_at_m, workspaces.deleted_at_m FROM `workspaces`
WHERE org_id = sqlc.arg(org_id)
AND deleted_at_m IS NULL;
