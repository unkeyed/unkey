-- name: ListOpenAlertEventGroups :many
-- ListOpenAlertEventGroups returns the durable groups that shards must keep
-- evaluating even when ClickHouse no longer classifies them as candidates.
SELECT DISTINCT
    workspace_id,
    project_id,
    app_id,
    environment_id
FROM alert_events
WHERE status = 'open'
ORDER BY workspace_id, project_id, app_id, environment_id;
