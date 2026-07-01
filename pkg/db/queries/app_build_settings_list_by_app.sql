-- name: ListAppBuildSettingsByApp :many
-- Returns the build settings for every environment in an app, for callers
-- that build multiple environments at once and group by environment_id.
SELECT *
FROM app_build_settings
WHERE app_id = sqlc.arg(app_id);
