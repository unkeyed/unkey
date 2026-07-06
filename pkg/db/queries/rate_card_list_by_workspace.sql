-- name: ListRateCardsByWorkspace :many
SELECT *
FROM rate_cards
WHERE workspace_id = sqlc.arg(workspace_id)
  AND archived = false
ORDER BY name;
