-- name: FindKeysWithoutCredits :many
SELECT 
    k.id,
    k.workspace_id,
    k.remaining_requests,
    k.refill_day,
    k.refill_amount,
    CASE 
        WHEN k.last_refill_at IS NULL THEN NULL 
        ELSE UNIX_TIMESTAMP(k.last_refill_at) * 1000 
    END as last_refill_at_unix,
    k.created_at_m,
    k.updated_at_m
FROM `keys` k
LEFT JOIN `credits` c ON c.key_id = k.id
WHERE k.deleted_at_m IS NULL
    AND k.remaining_requests IS NOT NULL
    AND c.id IS NULL
ORDER BY k.created_at_m DESC
LIMIT ?
OFFSET ?;