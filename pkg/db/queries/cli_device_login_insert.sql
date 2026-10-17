-- name: InsertCLIDeviceLogin :exec
INSERT INTO cli_device_logins (
    id,
    user_code,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(user_code),
    sqlc.arg(poll_interval_seconds),
    sqlc.arg(expires_at),
    sqlc.arg(status),
    sqlc.arg(device_name),
    sqlc.arg(requester_ip),
    sqlc.arg(requester_user_agent),
    sqlc.arg(created_at)
);
