-- name: HeartbeatLease :execrows
UPDATE leases 
SET heartbeat_at = ?,
    expires_at = ?
WHERE resource_id = ? AND worker_id = ?;

-- name: UpdateLease :exec
UPDATE leases 
SET worker_id = ?,
    acquired_at = ?,
    expires_at = ?,
    heartbeat_at = ?
WHERE resource_id = ?;