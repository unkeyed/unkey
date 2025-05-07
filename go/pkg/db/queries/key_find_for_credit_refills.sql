-- name: FindKeysForRefill :many
SELECT
  sqlc.embed(k),
  sqlc.embed(w)
FROM `keys` k
JOIN workspaces w ON k.workspace_id = w.id
WHERE
  k.deleted_at_m IS NULL
  AND k.refill_amount IS NOT NULL
  AND k.refill_amount > k.remaining_requests
  AND CASE
    WHEN sqlc.arg(is_last_day_of_month) THEN
      -- If today is last day of month, include keys with refill_day > today OR regular refill condition
      (k.refill_day > sqlc.arg(today) OR k.refill_day IS NULL OR k.refill_day = sqlc.arg(today))
    ELSE
      -- Otherwise, only include keys matching regular refill condition
      (k.refill_day IS NULL OR k.refill_day = sqlc.arg(today))
    END;
