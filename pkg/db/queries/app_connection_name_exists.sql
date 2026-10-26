-- name: ExistsAppConnectionName :one
SELECT EXISTS (
    SELECT 1 FROM app_connections
    WHERE workspace_id = sqlc.arg(workspace_id)
        AND app_id = sqlc.arg(app_id)
        AND environment_id = sqlc.arg(environment_id)
        AND name = sqlc.arg(name)
);
