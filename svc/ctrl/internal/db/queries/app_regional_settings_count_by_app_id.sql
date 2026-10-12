-- name: CountAppRegionalSettingsByAppId :one
-- CountAppRegionalSettingsByAppId counts an app's regional settings rows. Only tests use this query.
SELECT COUNT(*)
FROM `app_regional_settings`
WHERE app_id = sqlc.arg(app_id);
