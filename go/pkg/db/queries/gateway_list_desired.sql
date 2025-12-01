-- name: ListDesiredGateways :many
SELECT
    g.id as gateway_id,
    g.workspace_id,
    g.environment_id,
    g.k8s_service_name,
    g.region,
    g.image,
    g.desired_state,
    g.replicas,
    g.cpu_millicores,
    g.memory_mib,
    g.project_id,
    w.k8s_namespace as k8s_namespace
FROM `gateways` g
INNER JOIN `workspaces` w ON g.workspace_id = w.id
WHERE (sqlc.arg(region) = '' OR g.region = sqlc.arg(region))
    AND g.desired_state = sqlc.arg(desired_state)
    AND g.id > sqlc.arg(pagination_cursor)
ORDER BY g.id ASC
LIMIT ?;
