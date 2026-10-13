-- name: FindOldIdentityResourcesForUpdate :many
-- Locks the deleted duplicate and its ratelimits before cleanup so audit targets
-- describe the same resources the transaction deletes. The unique workspace,
-- external ID and deleted index permits at most one old identity.
SELECT i.id, i.external_id, rl.id AS ratelimit_id, rl.name AS ratelimit_name
FROM identities i
LEFT JOIN ratelimits rl ON i.id = rl.identity_id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.id != sqlc.arg(current_identity_id)
  AND i.deleted = true
FOR UPDATE;
