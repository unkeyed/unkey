-- name: UpsertOpenApiSpec :exec
INSERT INTO openapi_specs (workspace_id, deployment_id, portal_config_id, spec, created_at, updated_at)
VALUES (sqlc.arg(workspace_id), sqlc.narg(deployment_id), sqlc.narg(portal_config_id),
        sqlc.arg(spec), sqlc.arg(created_at), sqlc.arg(updated_at))
ON DUPLICATE KEY UPDATE
    spec = VALUES(spec),
    updated_at = VALUES(updated_at);
