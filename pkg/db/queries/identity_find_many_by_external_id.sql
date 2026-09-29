-- name: FindIdentitiesByExternalId :many
-- FindIdentitiesByExternalId resolves external IDs within one project, because
-- external IDs are unique per project.
SELECT identities.pk, identities.id, identities.external_id, identities.workspace_id, identities.project_id, identities.environment, identities.meta, identities.deleted, identities.created_at, identities.updated_at
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND external_id IN (sqlc.slice('externalIds'))
  AND deleted = sqlc.arg(deleted);
