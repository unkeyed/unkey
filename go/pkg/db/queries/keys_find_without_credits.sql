-- name: FindKeysWithoutCredits :many
SELECT
    k.id,
    k.workspace_id,
    k.remaining_requests,
    k.refill_day,
    k.refill_amount,
    k.last_refill_at,
    k.created_at_m
FROM `keys` k
LEFT JOIN `credits` c ON c.key_id = k.id
LEFT JOIN `credits` c2 ON c2.identity_id = k.identity_id
WHERE k.remaining_requests IS NOT NULL
    AND c.id IS NULL
    AND c2.id IS NULL
ORDER BY k.created_at_m DESC
LIMIT ?
OFFSET ?;
