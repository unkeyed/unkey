-- name: FindKeyCredits :one
SELECT remaining_requests FROM `keys` k WHERE k.id = ?;

-- name: FindLiveKeyCredits :one
SELECT remaining_requests
FROM `keys`
WHERE id = sqlc.arg(id)
  AND deleted_at_m IS NULL;

-- name: FindKeyCreditsBatchResult :one
-- This current locking read runs before COMMIT in the same multi-statement
-- command. outbox_rows mirrors whether the immediately preceding conditional
-- audit insert ran, and the key fields classify a guarded update miss.
SELECT
    ROW_COUNT() AS outbox_rows,
    k.id AS key_id,
    k.deleted_at_m,
    k.remaining_requests,
    k.refill_amount,
    k.refill_day
FROM (SELECT 1) singleton
LEFT JOIN `keys` k ON k.id = sqlc.arg(id)
FOR UPDATE;
