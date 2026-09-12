-- name: FindPendingDeployAnomalyEventMaxPk :one
SELECT CAST(COALESCE(MAX(pk), 0) AS UNSIGNED) AS max_pk
FROM deploy_anomaly_events
WHERE workspace_id = sqlc.arg(workspace_id)
    AND processed_at IS NULL;
