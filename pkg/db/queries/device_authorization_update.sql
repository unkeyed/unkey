-- name: ApproveDeviceAuthorization :execrows
UPDATE device_authorizations
SET
    status = 'approved',
    workspace_id = sqlc.arg(workspace_id),
    approver_user_id = sqlc.arg(approver_user_id),
    approver_name = sqlc.arg(approver_name),
    approver_roles = CAST(sqlc.arg(approver_roles) AS JSON),
    permissions = CAST(sqlc.arg(permissions) AS JSON),
    approved_at = sqlc.arg(approved_at),
    expires_at = sqlc.arg(expires_at),
    code_hash = sqlc.arg(code_hash)
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'approved')
  AND (
    approver_user_id IS NULL
    OR approver_user_id = sqlc.arg(approver_user_id)
  );

-- name: DenyDeviceAuthorization :execrows
UPDATE device_authorizations
SET status = 'denied'
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'approved');

-- name: ExpireDeviceAuthorization :execrows
UPDATE device_authorizations
SET status = 'expired'
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'approved');

-- name: ConsumeDeviceAuthorization :execrows
UPDATE device_authorizations
SET status = 'consumed', session_id = sqlc.arg(session_id)
WHERE id = sqlc.arg(id)
  AND status = 'approved';

-- name: DeleteExpiredDeviceAuthorizations :execrows
DELETE FROM device_authorizations
WHERE expires_at < sqlc.arg(expired_before)
LIMIT 100;
