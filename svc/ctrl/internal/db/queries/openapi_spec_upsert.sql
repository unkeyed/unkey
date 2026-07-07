-- name: UpsertOpenApiSpec :exec
INSERT INTO openapi_specs (id,workspace_id, deployment_id, portal_config_id, content, created_at, updated_at)
VALUES (sqlc.arg(id),sqlc.arg(workspace_id), sqlc.narg(deployment_id), sqlc.narg(portal_config_id),
        sqlc.arg(content), sqlc.arg(created_at), sqlc.arg(updated_at))
ON DUPLICATE KEY UPDATE
    content = VALUES(content),
    updated_at = VALUES(updated_at);
