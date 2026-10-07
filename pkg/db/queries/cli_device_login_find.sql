-- name: FindCLIDeviceLoginByID :one
SELECT
    pk,
    id,
    user_code,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    workspace_id,
    approver_user_id,
    approver_name,
    approver_roles,
    permissions,
    key_name,
    key_id,
    created_at,
    approved_at
FROM cli_device_logins
WHERE id = sqlc.arg(id);

-- name: FindCLIDeviceLoginByIDForUpdate :one
SELECT
    pk,
    id,
    user_code,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    workspace_id,
    approver_user_id,
    approver_name,
    approver_roles,
    permissions,
    key_name,
    key_id,
    created_at,
    approved_at
FROM cli_device_logins
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: FindCLIDeviceLoginByUserCode :one
SELECT
    pk,
    id,
    user_code,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    workspace_id,
    approver_user_id,
    approver_name,
    approver_roles,
    permissions,
    key_name,
    key_id,
    created_at,
    approved_at
FROM cli_device_logins
WHERE user_code = sqlc.arg(user_code);

-- name: FindCLIDeviceLoginByUserCodeForUpdate :one
SELECT
    pk,
    id,
    user_code,
    poll_interval_seconds,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    workspace_id,
    approver_user_id,
    approver_name,
    approver_roles,
    permissions,
    key_name,
    key_id,
    created_at,
    approved_at
FROM cli_device_logins
WHERE user_code = sqlc.arg(user_code)
FOR UPDATE;
