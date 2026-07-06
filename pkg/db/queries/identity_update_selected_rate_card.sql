-- name: UpdateIdentitySelectedRateCard :exec
UPDATE identities
SET selected_rate_card_id = sqlc.arg(selected_rate_card_id)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(identity_id)
  AND deleted = false;
