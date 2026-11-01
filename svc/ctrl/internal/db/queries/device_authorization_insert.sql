-- name: InsertDeviceAuthorization :exec
-- Inserts an approved device authorization. Used only by tests; the API owns
-- the device flow.
INSERT INTO device_authorizations (id, user_code, expires_at, status, workspace_id, approver_user_id, approver_name, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_code), sqlc.arg(expires_at), 'approved', sqlc.arg(workspace_id), sqlc.arg(approver_user_id), sqlc.arg(approver_name), sqlc.arg(created_at));
