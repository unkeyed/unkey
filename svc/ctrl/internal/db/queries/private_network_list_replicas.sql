-- name: ListPrivateNetworkReplicas :many
-- ListPrivateNetworkReplicas returns the given deployments that are active,
-- were created with private networking, and run on the platform. Each one
-- resolves its own replicas under its app's slug and is a caller for
-- ListPrivateNetworkConnections.
-- NO_SEMIJOIN keeps the topology check per deployment; as a semijoin MySQL may
-- read every running topology row first.
SELECT
    d.workspace_id,
    d.project_id,
    d.app_id,
    a.slug AS app_slug,
    w.k8s_namespace,
    d.id AS deployment_id,
    d.port,
    d.environment_id
FROM deployments d
STRAIGHT_JOIN apps a ON a.id = d.app_id
    AND a.workspace_id = d.workspace_id AND a.project_id = d.project_id
STRAIGHT_JOIN environments e ON e.id = d.environment_id AND e.app_id = d.app_id
STRAIGHT_JOIN workspaces w ON w.id = d.workspace_id
WHERE d.id IN (sqlc.slice(deployment_ids))
    AND d.desired_state = 'running'
    AND d.status IN ('deploying', 'network', 'finalizing', 'ready')
    AND JSON_CONTAINS(d.capabilities, 'true', '$.private_networking')
    AND w.k8s_namespace <> ''
    AND EXISTS (
        SELECT /*+ NO_SEMIJOIN() */ 1 FROM deployment_topology dt
        INNER JOIN regions r ON r.id = dt.region_id
        WHERE dt.deployment_id = d.id
            AND dt.desired_status = 'running'
            AND r.platform = sqlc.arg(platform)
    );
