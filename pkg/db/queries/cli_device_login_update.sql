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

-- name: UpdateCLIDeviceLoginInterval :exec
UPDATE cli_device_logins
SET poll_interval_seconds = sqlc.arg(poll_interval_seconds)
WHERE id = sqlc.arg(id);

-- name: UpdateCLIDeviceLoginStatus :exec
UPDATE cli_device_logins
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id);

-- name: ConsumeCLIDeviceLogin :execrows
UPDATE cli_device_logins
SET status = 'consumed'
WHERE id = sqlc.arg(id)
  AND status = 'approved';
