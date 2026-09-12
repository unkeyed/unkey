-- name: InsertDeployAnomalyEvent :exec
INSERT INTO deploy_anomaly_events (
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    deployment_id,
    metric,
    event_time,
    received_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE id = id;
