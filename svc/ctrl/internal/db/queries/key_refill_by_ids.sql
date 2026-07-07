-- name: RefillKeysByIDs :exec
-- RefillKeysByIDs sets remaining_requests to refill_amount for the given keys.
-- This is a bulk operation to minimize database round trips.
UPDATE `keys`
SET remaining_requests = refill_amount,
    last_refill_at = NOW(3),
    updated_at_m = sqlc.arg(now)
WHERE id IN (sqlc.slice(ids))
  AND deleted_at_m IS NULL;
