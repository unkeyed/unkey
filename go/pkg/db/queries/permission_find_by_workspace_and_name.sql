-- name: FindPermissionByWorkspaceAndName :one
SELECT * FROM `permissions`
WHERE workspace_id = sqlc.arg(workspace_id) AND name = sqlc.arg(name);
