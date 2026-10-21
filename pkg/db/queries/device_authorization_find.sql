-- name: FindDeviceAuthorizationByID :one
SELECT
    pk,
    id,
    user_code,
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
    session_id,
    created_at,
    approved_at
FROM device_authorizations
WHERE id = sqlc.arg(id);

-- name: FindDeviceAuthorizationByIDForUpdate :one
SELECT
    pk,
    id,
    user_code,
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
    session_id,
    created_at,
    approved_at
FROM device_authorizations
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: FindDeviceAuthorizationByUserCode :one
SELECT
    pk,
    id,
    user_code,
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
    session_id,
    created_at,
    approved_at
FROM device_authorizations
WHERE user_code = sqlc.arg(user_code);

-- name: FindDeviceAuthorizationByUserCodeForUpdate :one
SELECT
    pk,
    id,
    user_code,
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
    session_id,
    created_at,
    approved_at
FROM device_authorizations
WHERE user_code = sqlc.arg(user_code)
FOR UPDATE;
