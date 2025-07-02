-- name: AcquireLease :exec
INSERT INTO leases (
    resource_id,
    kind,
    namespace,
    worker_id,
    acquired_at,
    expires_at,
    heartbeat_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);