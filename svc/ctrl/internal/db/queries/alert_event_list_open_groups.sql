-- name: ListOpenAlertEventGroups :many
-- ListOpenAlertEventGroups returns the durable groups that shards must keep
-- evaluating even when ClickHouse no longer classifies them as candidates.
-- workspace_hash uses the same CityHash64 function as ClickHouse sharding.
SELECT DISTINCT
    workspace_id,
    project_id,
    app_id,
    environment_id
FROM alert_events
WHERE status = 'open'
  AND workspace_hash % sqlc.arg(shard_count) = sqlc.arg(shard)
ORDER BY workspace_id, project_id, app_id, environment_id;
