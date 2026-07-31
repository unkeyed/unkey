-- name: ListAllDeploymentTopologiesByRegion :many
-- ListAllDeploymentTopologiesByRegion returns running deployment topologies for a region, paginated by pk.
-- Used by SyncDesiredState to reconcile krane agents with current desired state.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace,
    e.slug AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON d.id = dt.deployment_id
INNER JOIN `workspaces` w ON w.id = d.workspace_id
INNER JOIN `regions` r ON r.id = dt.region_id
INNER JOIN `environments` e ON e.id = d.environment_id
LEFT JOIN `github_repo_connections` grc ON grc.app_id = d.app_id
WHERE r.id = sqlc.arg(region_id) AND dt.pk > sqlc.arg(after_pk) AND dt.desired_status = 'running'
ORDER BY dt.pk ASC
LIMIT ?;
