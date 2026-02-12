-- name: ListKeysForRefill :many
-- ListKeysForRefill returns keys that need their remaining_requests refilled.
-- Uses a deferred join on pk for stable cursor-based pagination that avoids
-- OFFSET drift when rows are mutated between batches.
-- Keys are selected if:
--   - refill_day is NULL (daily refill)
--   - refill_day matches today's day of month
--   - refill_day > today's day AND today is the last day of month (catch-up for short months)
-- Keys are skipped if remaining_requests >= refill_amount (already full).
SELECT k.pk, k.id, k.workspace_id, k.refill_amount, k.remaining_requests, k.name
FROM `keys` k
INNER JOIN (
    SELECT ki.pk
    FROM `keys` ki
    WHERE ki.refill_amount IS NOT NULL
      AND ki.deleted_at_m IS NULL
      AND (ki.remaining_requests IS NULL OR ki.refill_amount > ki.remaining_requests)
      AND (
          ki.refill_day IS NULL
          OR ki.refill_day = sqlc.arg(today_day)
          OR (sqlc.arg(is_last_day_of_month) = 1 AND ki.refill_day > sqlc.arg(today_day))
      )
      AND ki.pk > sqlc.arg(after_pk)
    ORDER BY pk
    LIMIT ?
) AS batch ON batch.pk = k.pk;
