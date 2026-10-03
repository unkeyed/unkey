-- name: ListPrivateNetworkReplicas :many
-- ListPrivateNetworkReplicas returns one page of active deployments in
-- workspaces enrolled in private networking, ordered by deployment ID. Each
-- deployment resolves its own replicas under its app's slug. A workspace is
-- enrolled while it has at least one app binding to another app.
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
INNER JOIN apps a ON a.id = d.app_id
    AND a.workspace_id = d.workspace_id AND a.project_id = d.project_id
INNER JOIN environments e ON e.id = d.environment_id AND e.app_id = d.app_id
INNER JOIN workspaces w ON w.id = d.workspace_id AND w.k8s_namespace <> ''
WHERE d.id > sqlc.arg(after_deployment_id)
    AND d.status IN ('deploying', 'ready') AND d.desired_state = 'running'
    AND EXISTS (
        SELECT 1 FROM app_bindings b
        WHERE b.workspace_id = d.workspace_id
            AND b.resource_type = 'app'
            AND b.resource_id <> b.app_id
    )
    AND EXISTS (
        SELECT 1 FROM deployment_topology dt
        INNER JOIN regions r ON r.id = dt.region_id
        WHERE dt.deployment_id = d.id
            AND dt.desired_status = 'running'
            AND r.platform = sqlc.arg(platform)
    )
ORDER BY d.id
LIMIT ?;
