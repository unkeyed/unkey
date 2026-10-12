-- name: CountAppBuildSettingsByAppId :one
-- CountAppBuildSettingsByAppId counts an app's build settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_build_settings`
WHERE app_id = sqlc.arg(app_id);
