-- name: InsertCredit :exec
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
