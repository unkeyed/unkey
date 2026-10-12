-- name: CountAppRuntimeSettingsByWorkspaceId :one
-- CountAppRuntimeSettingsByWorkspaceId counts a workspace's runtime settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_runtime_settings`
WHERE workspace_id = sqlc.arg(workspace_id);
