-- name: DeleteIdentity :exec
DELETE FROM identities
WHERE id = sqlc.arg(identity_id)
  AND workspace_id = sqlc.arg(workspace_id);
