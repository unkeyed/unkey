-- name: ListSelectableRateCards :many
SELECT *
FROM rate_cards
WHERE workspace_id = sqlc.arg(workspace_id)
  AND selectable = true
  AND archived = false
ORDER BY name;
