-- name: ListOpenAlertEventGroups :many
SELECT DISTINCT
    workspace_id,
    project_id,
    app_id,
    environment_id
FROM alert_events
WHERE status = 'open'
ORDER BY workspace_id, project_id, app_id, environment_id;
