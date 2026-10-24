-- name: RefillKeysByIDs :execrows
-- RefillKeysByIDs sets remaining_requests to refill_amount for the given keys.
-- The journaled refill time prevents a committed refill from running again.
-- Call with one ID to determine whether that key needs an audit event.
UPDATE `keys`
SET remaining_requests = refill_amount,
    last_refill_at = sqlc.arg(refill_at),
    updated_at_m = sqlc.arg(now)
WHERE id IN (sqlc.slice(ids))
  AND deleted_at_m IS NULL
  AND (last_refill_at IS NULL OR last_refill_at < sqlc.arg(refill_at));
