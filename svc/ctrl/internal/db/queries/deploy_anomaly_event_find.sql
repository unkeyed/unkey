-- name: FindDeployAnomalyEvent :one
SELECT
    pk,
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    deployment_id,
    metric,
    event_time,
    received_at,
    processed_at
FROM deploy_anomaly_events
WHERE id = sqlc.arg(id)
    AND workspace_id = sqlc.arg(workspace_id)
FOR UPDATE;
