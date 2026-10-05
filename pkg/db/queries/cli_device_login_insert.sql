-- name: InsertCLIDeviceLogin :exec
INSERT INTO cli_device_logins (
    id,
    user_code,
    device_code,
    workos_verification_uri,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(user_code),
    sqlc.arg(device_code),
    sqlc.arg(workos_verification_uri),
    sqlc.arg(poll_interval_seconds),
    sqlc.arg(expires_at),
    sqlc.arg(status),
    sqlc.arg(device_name),
    sqlc.arg(created_at)
);
