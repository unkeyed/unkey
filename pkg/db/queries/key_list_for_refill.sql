-- name: ListKeysForRefill :many
-- ListKeysForRefill returns keys that need their remaining_requests refilled.
-- Uses idx_keys_refill index for efficient lookup.
-- Keys are selected if:
--   - refill_day is NULL (daily refill)
--   - refill_day matches today's day of month
--   - refill_day > today's day AND today is the last day of month (catch-up for short months)
-- Keys are skipped if remaining_requests >= refill_amount (already full).
SELECT id, workspace_id, refill_amount, remaining_requests, name
FROM `keys`
WHERE refill_amount IS NOT NULL
  AND deleted_at_m IS NULL
  AND (remaining_requests IS NULL OR refill_amount > remaining_requests)
  AND (
      refill_day IS NULL
      OR refill_day = sqlc.arg(today_day)
      OR (sqlc.arg(is_last_day_of_month) = 1 AND refill_day > sqlc.arg(today_day))
  )
ORDER BY id
LIMIT ? OFFSET ?;
