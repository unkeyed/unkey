-- name: FindOpenAlertEventsByGroup :many
SELECT
    pk,
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    deployment_id,
    metric,
    status,
    fired_at,
    last_seen_at,
    resolved_at,
    resolved_by,
    resolution_message,
    observed_value,
    baseline_mean,
    baseline_stddev,
    threshold_sigma,
    window_start,
    window_end,
    created_at,
    updated_at
FROM alert_events
WHERE workspace_id = sqlc.arg(workspace_id)
  AND app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND status = 'open'
ORDER BY metric ASC;
