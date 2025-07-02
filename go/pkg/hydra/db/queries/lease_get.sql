-- name: GetLease :one
SELECT * FROM leases 
WHERE resource_id = ?;

-- name: GetExpiredLeases :many
SELECT * FROM leases 
WHERE namespace = ? AND expires_at < ?;