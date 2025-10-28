-- name: UpsertCredit :exec
INSERT INTO `credits` (
    id,
    workspace_id,
    key_id,
    identity_id,
    remaining,
    refill_day,
    refill_amount,
    created_at,
    updated_at,
    refilled_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    remaining = CASE
        WHEN CAST(sqlc.arg('remaining_specified') AS UNSIGNED) = 1 THEN VALUES(remaining)
        ELSE remaining
    END,
    refill_day = CASE
        WHEN CAST(sqlc.arg('refill_day_specified') AS UNSIGNED) = 1 THEN VALUES(refill_day)
        ELSE refill_day
    END,
    refill_amount = CASE
        WHEN CAST(sqlc.arg('refill_amount_specified') AS UNSIGNED) = 1 THEN VALUES(refill_amount)
        ELSE refill_amount
    END,
    updated_at = VALUES(updated_at);
