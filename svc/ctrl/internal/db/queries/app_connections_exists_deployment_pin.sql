-- name: ExistsAppConnectionPinningDeployment :one
SELECT EXISTS(
    SELECT 1 FROM app_connections
    WHERE resource_type = 'app'
        AND selection_mode = 'deployment'
        AND target_deployment_id = sqlc.arg(deployment_id)
) AS pinned;
