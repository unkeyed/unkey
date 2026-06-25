-- name: SoftDeleteIdentity :exec
UPDATE identities
SET deleted = 1
WHERE id = sqlc.arg(identity_id)
  AND workspace_id = sqlc.arg(workspace_id);
