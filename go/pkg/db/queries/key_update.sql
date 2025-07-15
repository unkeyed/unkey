-- name: UpdateKey :exec
UPDATE `keys` k SET
name = COALESCE(sqlc.narg('name'), k.name),
owner_id = sqlc.narg('owner_id'),
identity_id = sqlc.narg('identity_id'),
enabled = COALESCE(sqlc.narg('enabled'), k.enabled),
meta = COALESCE(sqlc.narg('meta'), k.meta),
expires = COALESCE(sqlc.narg('expires'), k.expires),
remaining_requests = COALESCE(sqlc.narg('remaining_requests'), k.remaining_requests),
refill_amount = COALESCE(sqlc.narg('refill_amount'), k.refill_amount),
refill_day = COALESCE(sqlc.narg('refill_day'), k.refill_day),
updated_at_m = sqlc.arg('now')
WHERE id = sqlc.arg('id');
