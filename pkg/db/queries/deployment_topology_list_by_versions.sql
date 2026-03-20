-- name: ListDeploymentTopologyByRegion :many
-- ListDeploymentTopologyByRegion returns deployment topologies for a region with version > after_version.
-- Used by WatchDeployments to stream deployment state changes to krane agents.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace,
    p.slug AS project_slug,
    a.slug AS app_slug,
    e.slug AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
INNER JOIN `regions` r ON dt.region_id = r.id
INNER JOIN `projects` p ON d.project_id = p.id
INNER JOIN `apps` a ON d.app_id = a.id
INNER JOIN `environments` e ON d.environment_id = e.id
LEFT JOIN `github_repo_connections` grc ON d.app_id = grc.app_id
WHERE r.id = sqlc.arg(region_id) AND dt.version > sqlc.arg(afterVersion)
ORDER BY dt.version ASC
LIMIT ?;
