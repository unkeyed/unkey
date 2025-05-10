-- name: RefillKey :exec
UPDATE `keys`
SET
  remaining_requests = sqlc.arg(refill_amount),
  last_refill_at = sqlc.arg(now)
  WHERE id = sqlc.arg(key_id);
