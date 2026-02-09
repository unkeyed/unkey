-- name: FindEnvironmentBuildSettingsByEnvironmentId :one
SELECT *
FROM environment_build_settings
WHERE environment_id = ?;
