-- name: ResolveOpenAlertEventsByGroups :exec
-- ResolveOpenAlertEventsByGroups self-heals detector alerts whose app or
-- environment metadata was removed before lifecycle cleanup could close them.
-- The JSON batch matches the metadata lookup boundary, limiting each call to
-- at most 500 groups.
WITH RECURSIVE group_indexes AS (
    SELECT 0 AS group_index
    WHERE JSON_LENGTH(sqlc.arg(group_keys_json)) > 0
    UNION ALL
    SELECT group_index + 1
    FROM group_indexes
    WHERE group_index + 1 < JSON_LENGTH(sqlc.arg(group_keys_json))
), requested AS (
    SELECT
        JSON_UNQUOTE(JSON_EXTRACT(sqlc.arg(group_keys_json), CONCAT('$[', group_index, '].workspace_id'))) AS workspace_id,
        JSON_UNQUOTE(JSON_EXTRACT(sqlc.arg(group_keys_json), CONCAT('$[', group_index, '].project_id'))) AS project_id,
        JSON_UNQUOTE(JSON_EXTRACT(sqlc.arg(group_keys_json), CONCAT('$[', group_index, '].app_id'))) AS app_id,
        JSON_UNQUOTE(JSON_EXTRACT(sqlc.arg(group_keys_json), CONCAT('$[', group_index, '].environment_id'))) AS environment_id
    FROM group_indexes
)
UPDATE alert_events AS ae
INNER JOIN requested
    ON requested.workspace_id = ae.workspace_id
    AND requested.project_id = ae.project_id
    AND requested.app_id = ae.app_id
    AND requested.environment_id = ae.environment_id
SET ae.status = 'resolved',
    ae.resolved_at = sqlc.arg(resolved_at),
    ae.resolution_message = 'Deployment stopped',
    ae.updated_at = sqlc.arg(updated_at)
WHERE ae.status = 'open';
