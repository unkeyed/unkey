-- name: DeleteOldIdentityByExternalID :exec
-- DeleteOldIdentityByExternalID hard-deletes the soft-deleted identity that
-- blocks a new soft delete on the per-project unique key. It is scoped to the
-- project so identities with the same external ID in other projects survive.
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON i.id = rl.identity_id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.project_id = sqlc.arg(project_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.id != sqlc.arg(current_identity_id)
  AND i.deleted = true;
