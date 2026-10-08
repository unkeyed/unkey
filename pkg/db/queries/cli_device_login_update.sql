-- name: ApproveCLIDeviceLogin :execrows
UPDATE cli_device_logins
SET
    status = 'approved',
    workspace_id = sqlc.arg(workspace_id),
    approver_user_id = sqlc.arg(approver_user_id),
    approver_name = sqlc.arg(approver_name),
    approver_roles = CAST(sqlc.arg(approver_roles) AS JSON),
    permissions = CAST(sqlc.arg(permissions) AS JSON),
    key_name = sqlc.arg(key_name),
    approved_at = sqlc.arg(approved_at)
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'approved')
  AND (
    approver_user_id IS NULL
    OR approver_user_id = sqlc.arg(approver_user_id)
  );

-- name: UpdateCLIDeviceLoginStatus :exec
UPDATE cli_device_logins
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id);

-- name: ConsumeCLIDeviceLogin :execrows
UPDATE cli_device_logins
SET status = 'consumed', key_id = sqlc.arg(key_id)
WHERE id = sqlc.arg(id)
  AND status = 'approved';

-- name: DeleteExpiredCLIDeviceLogins :execrows
DELETE FROM cli_device_logins
WHERE expires_at < sqlc.arg(expired_before)
LIMIT 100;
