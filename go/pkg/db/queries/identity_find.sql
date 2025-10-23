-- name: FindIdentity :one
SELECT
  identities.*,
  credits.id AS credit_id,
  credits.remaining AS credit_remaining,
  credits.refill_amount AS credit_refill_amount,
  credits.refill_day AS credit_refill_day,
  credits.refilled_at AS credit_refilled_at
FROM identities
LEFT JOIN credits ON credits.identity_id = identities.id
WHERE identities.workspace_id = sqlc.arg(workspace_id)
  AND (identities.external_id = sqlc.arg(identity) OR identities.id = sqlc.arg(identity))
  AND identities.deleted = sqlc.arg(deleted);
