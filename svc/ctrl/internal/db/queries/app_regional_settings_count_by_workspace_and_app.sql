-- name: CountAppRegionalSettingsByWorkspaceAndApp :one
-- CountAppRegionalSettingsByWorkspaceAndApp counts an app's regional settings rows within a workspace. Only tests use this query.
SELECT COUNT(*)
FROM `app_regional_settings`
WHERE workspace_id = sqlc.arg(workspace_id) AND app_id = sqlc.arg(app_id);
