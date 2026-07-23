-- name: FindAppDockerSourceByAppId :one
SELECT
    pk,
    workspace_id,
    app_id,
    image,
    created_at,
    updated_at
FROM app_docker_sources
WHERE app_id = sqlc.arg(app_id);
