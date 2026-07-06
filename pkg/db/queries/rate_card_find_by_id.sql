-- name: FindRateCardByID :one
SELECT *
FROM rate_cards
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(rate_card_id);
