-- name: FindEnvironmentRuntimeSettingsByEnvironmentId :one
SELECT *
FROM environment_runtime_settings
WHERE environment_id = ?;
