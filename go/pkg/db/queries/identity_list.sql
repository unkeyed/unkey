-- name: ListIdentities :many
SELECT 
    i.*,
    c.id as credit_id,
    c.remaining as credit_remaining,
    c.refill_amount as credit_refill_amount,
    c.refill_day as credit_refill_day
FROM identities i
LEFT JOIN credits c ON c.identity_id = i.id
WHERE i.workspace_id = sqlc.arg(workspace_id)
AND i.deleted = sqlc.arg(deleted)
AND i.id >= sqlc.arg(id_cursor)
ORDER BY i.id ASC
LIMIT ?
