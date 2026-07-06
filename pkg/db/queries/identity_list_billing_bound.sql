-- name: ListBillingBoundIdentities :many
-- Identities whose usage is pushed to a billing provider at period close.
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND billing_provider = sqlc.arg(billing_provider)
  AND deleted = false
ORDER BY pk
LIMIT ? OFFSET ?;
