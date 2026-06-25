-- name: DeleteOldIdentityByExternalID :exec
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON rl.identity_id = i.id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.id != sqlc.arg(current_identity_id)
  AND i.deleted = true;
