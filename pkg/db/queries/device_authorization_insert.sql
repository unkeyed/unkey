-- name: InsertDeviceAuthorization :exec
INSERT INTO device_authorizations (
    id,
    user_code,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(user_code),
    sqlc.arg(expires_at),
    sqlc.arg(status),
    sqlc.arg(device_name),
    sqlc.arg(requester_ip),
    sqlc.arg(requester_user_agent),
    sqlc.arg(created_at)
);
