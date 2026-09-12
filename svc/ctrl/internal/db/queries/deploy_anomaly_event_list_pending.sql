-- name: ListPendingDeployAnomalyEvents :many
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
WHERE workspace_id = sqlc.arg(workspace_id)
    AND processed_at IS NULL
    AND pk > sqlc.arg(after_pk)
    AND pk <= sqlc.arg(through_pk)
ORDER BY pk
LIMIT ?;
