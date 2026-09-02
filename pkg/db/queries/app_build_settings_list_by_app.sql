-- name: ListAppBuildSettingsByApp :many
-- Returns the build settings for every environment in an app, for callers
-- that build multiple environments at once and group by environment_id.
SELECT app_build_settings.pk, app_build_settings.workspace_id, app_build_settings.app_id, app_build_settings.environment_id, app_build_settings.dockerfile, app_build_settings.docker_context, app_build_settings.build_command, app_build_settings.watch_paths, app_build_settings.auto_deploy, app_build_settings.created_at, app_build_settings.updated_at
FROM app_build_settings
WHERE app_id = sqlc.arg(app_id);
