-- name: CountAppBuildSettingsByWorkspaceId :one
-- CountAppBuildSettingsByWorkspaceId counts a workspace's build settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_build_settings`
WHERE workspace_id = sqlc.arg(workspace_id);
