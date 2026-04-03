-- name: ListAllDeploymentTopologiesByRegion :many
-- ListAllDeploymentTopologiesByRegion returns deployment topologies for a region, paginated by pk.
-- Used during full sync (version=0) to bootstrap krane agents with current state.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace,
    e.slug AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo,
    ars.replicas AS regional_replicas
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
INNER JOIN `regions` r ON dt.region_id = r.id
INNER JOIN `environments` e ON d.environment_id = e.id
LEFT JOIN `github_repo_connections` grc ON d.app_id = grc.app_id
LEFT JOIN `app_regional_settings` ars ON ars.app_id = d.app_id AND ars.environment_id = d.environment_id AND ars.region_id = dt.region_id
WHERE r.id = sqlc.arg(region_id) AND dt.pk > sqlc.arg(after_pk)
ORDER BY dt.pk ASC
LIMIT ?;
