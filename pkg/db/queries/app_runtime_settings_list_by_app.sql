-- name: ListAppRuntimeSettingsByApp :many
-- Returns the runtime settings for every environment in an app, for callers
-- that build multiple environments at once and group by environment_id.
SELECT sqlc.embed(app_runtime_settings)
FROM app_runtime_settings
WHERE app_id = sqlc.arg(app_id);
