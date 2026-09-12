-- name: MarkDeployAnomalyEventProcessed :exec
UPDATE deploy_anomaly_events
SET processed_at = sqlc.arg(processed_at)
WHERE id = sqlc.arg(id)
    AND workspace_id = sqlc.arg(workspace_id)
    AND processed_at IS NULL;
