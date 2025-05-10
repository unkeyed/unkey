-- name: FindKeysForRefill :many
SELECT
  *
FROM `keys`
WHERE
  deleted_at_m IS NULL
  AND refill_amount IS NOT NULL
  AND refill_amount > remaining_requests
  AND (
    last_refill_at < sqlc.arg(cutoff)
    OR last_refill_at IS NULL
  )
  AND CASE
    WHEN sqlc.arg(is_last_day_of_month) THEN
      -- If today is last day of month, include keys with refill_day > today OR regular refill condition
      (refill_day > sqlc.arg(today) OR refill_day IS NULL OR refill_day = sqlc.arg(today))
    ELSE
      -- Otherwise, only include keys matching regular refill condition
      (refill_day IS NULL OR refill_day = sqlc.arg(today))
    END;
