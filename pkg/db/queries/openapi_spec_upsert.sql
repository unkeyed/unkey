-- name: UpsertOpenApiSpec :exec
INSERT INTO openapi_specs (workspace_id, app_id, project_id, deployment_id, spec, created_at, updated_at)
VALUES (sqlc.arg(workspace_id), sqlc.narg(app_id), sqlc.arg(project_id),
        sqlc.arg(deployment_id), sqlc.arg(spec), sqlc.arg(created_at), sqlc.arg(updated_at))
ON DUPLICATE KEY UPDATE
    spec = VALUES(spec),
    updated_at = VALUES(updated_at);
