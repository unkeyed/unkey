-- name: FindIdentityByID :one
SELECT identities.pk, identities.id, identities.external_id, identities.workspace_id, identities.project_id, identities.environment, identities.meta, identities.deleted, identities.created_at, identities.updated_at
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(identity_id)
  AND deleted = sqlc.arg(deleted);
