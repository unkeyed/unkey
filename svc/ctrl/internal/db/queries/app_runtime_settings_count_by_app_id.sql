-- name: CountAppRuntimeSettingsByAppId :one
-- CountAppRuntimeSettingsByAppId counts an app's runtime settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_runtime_settings`
WHERE app_id = sqlc.arg(app_id);
