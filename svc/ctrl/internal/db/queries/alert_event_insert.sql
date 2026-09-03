-- name: InsertAlertEvent :exec
-- InsertAlertEvent persists the opening snapshot used for alert display and
-- hysteretic recovery. The evaluator checks for an existing open metric in the
-- same journaled step before calling this query.
INSERT INTO alert_events (
    id,
    workspace_id,
    workspace_hash,
    project_id,
    app_id,
    environment_id,
    deployment_id,
    metric,
    status,
    fired_at,
    last_seen_at,
    observed_value,
    baseline_mean,
    baseline_stddev,
    threshold_sigma,
    window_start,
    window_end,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(workspace_hash),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.narg(deployment_id),
    sqlc.arg(metric),
    'open',
    sqlc.arg(fired_at),
    sqlc.arg(last_seen_at),
    sqlc.arg(observed_value),
    sqlc.arg(baseline_mean),
    sqlc.arg(baseline_stddev),
    sqlc.arg(threshold_sigma),
    sqlc.arg(window_start),
    sqlc.arg(window_end),
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
);
