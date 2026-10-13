-- name: DeleteOldIdentityByExternalID :execrows
-- Removes the deleted duplicate and its ratelimits. Callers must first lock
-- these resources and compare the affected count with the captured audit targets.
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON i.id = rl.identity_id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.id != sqlc.arg(current_identity_id)
  AND i.deleted = true;
