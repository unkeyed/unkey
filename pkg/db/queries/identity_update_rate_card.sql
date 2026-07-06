-- name: UpdateIdentityRateCard :exec
UPDATE identities
SET rate_card_id = sqlc.arg(rate_card_id)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(identity_id)
  AND deleted = false;
