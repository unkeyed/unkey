-- name: ReleaseLease :execrows
DELETE FROM leases 
WHERE resource_id = ? AND worker_id = ?;

-- name: CleanupExpiredLeases :exec
DELETE FROM leases 
WHERE namespace = ? AND expires_at < ?;