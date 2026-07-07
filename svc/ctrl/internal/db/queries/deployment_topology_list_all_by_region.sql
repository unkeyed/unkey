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
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
INNER JOIN `regions` r ON dt.region_id = r.id
INNER JOIN `environments` e ON d.environment_id = e.id
LEFT JOIN `github_repo_connections` grc ON d.app_id = grc.app_id
WHERE r.id = sqlc.arg(region_id) AND dt.pk > sqlc.arg(after_pk) AND dt.desired_status = 'running'
ORDER BY dt.pk ASC
LIMIT ?;
