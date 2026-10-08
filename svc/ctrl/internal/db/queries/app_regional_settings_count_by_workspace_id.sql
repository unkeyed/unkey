-- name: CountAppRegionalSettingsByWorkspaceId :one
-- CountAppRegionalSettingsByWorkspaceId counts a workspace's regional settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_regional_settings`
WHERE workspace_id = sqlc.arg(workspace_id);
