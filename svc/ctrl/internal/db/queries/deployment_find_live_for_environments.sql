-- name: FindLiveDeploymentsForEnvironments :many
-- FindLiveDeploymentsForEnvironments resolves alert metadata in bounded
-- batches. The JSON array preserves complete four-column group identities;
-- ordering makes shard fan-out deterministic after missing groups are omitted.
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
SELECT
    e.workspace_id,
    e.project_id,
    e.app_id,
    e.id AS environment_id,
    w.org_id,
    w.name AS workspace_name,
    w.slug AS workspace_slug,
    a.name AS app_name,
    e.kind AS environment_kind,
    e.slug AS environment_slug,
    d.id AS deployment_id,
    COALESCE(d.desired_state, '') AS deployment_desired_state
FROM requested
INNER JOIN environments e
    ON BINARY e.id = BINARY requested.environment_id
    AND BINARY e.workspace_id = BINARY requested.workspace_id
    AND BINARY e.project_id = BINARY requested.project_id
    AND BINARY e.app_id = BINARY requested.app_id
INNER JOIN apps a ON a.id = e.app_id
INNER JOIN workspaces w ON w.id = e.workspace_id
LEFT JOIN deployments d
    ON d.id = a.current_deployment_id
    AND d.environment_id = e.id
WHERE w.deleted_at_m IS NULL
ORDER BY requested.workspace_id, requested.project_id, requested.app_id, requested.environment_id;
