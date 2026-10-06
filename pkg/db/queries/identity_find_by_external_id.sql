-- name: FindIdentityByExternalID :one
-- FindIdentityByExternalID resolves an external ID within one project. External
-- IDs are unique per project, so the same value can name different identities
-- in other projects of the workspace.
SELECT identities.pk, identities.id, identities.external_id, identities.workspace_id, identities.project_id, identities.environment, identities.meta, identities.deleted, identities.created_at, identities.updated_at
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND external_id = sqlc.arg(external_id)
  AND deleted = sqlc.arg(deleted);
