-- name: UpdateAppBuildSettings :exec
-- Updates only the columns whose *_specified flag is 1, preserving all others.
-- columns cannot overwrite each other. dockerfile is clearable (narg -> NULL).
UPDATE app_build_settings t
SET
    dockerfile = CASE
        WHEN CAST(sqlc.arg('dockerfile_specified') AS UNSIGNED) = 1 THEN sqlc.narg('dockerfile')
        ELSE t.dockerfile
    END,
    docker_context = CASE
        WHEN CAST(sqlc.arg('docker_context_specified') AS UNSIGNED) = 1 THEN sqlc.arg('docker_context')
        ELSE t.docker_context
    END,
    watch_paths = CASE
        WHEN CAST(sqlc.arg('watch_paths_specified') AS UNSIGNED) = 1 THEN sqlc.arg('watch_paths')
        ELSE t.watch_paths
    END,
    auto_deploy = CASE
        WHEN CAST(sqlc.arg('auto_deploy_specified') AS UNSIGNED) = 1 THEN sqlc.arg('auto_deploy')
        ELSE t.auto_deploy
    END,
    updated_at = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND app_id = sqlc.arg('app_id')
  AND environment_id = sqlc.arg('environment_id');
