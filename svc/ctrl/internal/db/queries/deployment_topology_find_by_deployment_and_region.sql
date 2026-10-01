-- name: FindDeploymentTopologyByDeploymentAndRegion :one
-- FindDeploymentTopologyByDeploymentAndRegion returns a single deployment topology with all
-- joined data needed for the Watch stream. Used by the unified WatchDeploymentChanges RPC.
SELECT
    dt.desired_status,
    CAST((p.id IS NULL OR a.id IS NULL OR e.id IS NULL
      OR p.deleting_at IS NOT NULL OR a.deleting_at IS NOT NULL OR e.deleting_at IS NOT NULL) AS SIGNED) AS removal_required,
    CAST((dt.desired_status = 'stopped' AND (
      (d.desired_state = 'stopped' AND d.status <> 'stopped')
      OR EXISTS (SELECT 1 FROM instances i WHERE i.deployment_id = dt.deployment_id AND i.region_id = dt.region_id)
    )) AS SIGNED) AS status_repair_required,
    dt.autoscaling_replicas_min,
    dt.autoscaling_replicas_max,
    dt.autoscaling_threshold_cpu,
    dt.autoscaling_threshold_memory,
    d.id,
    d.k8s_name,
    d.workspace_id,
    d.project_id,
    d.environment_id,
    d.app_id,
    d.image_resolved,
    d.build_id,
    d.git_commit_sha,
    d.git_branch,
    d.git_commit_message,
    d.cpu_millicores,
    d.memory_mib,
    d.storage_mib,
    d.encrypted_environment_variables,
    d.command,
    d.port,
    d.shutdown_signal,
    d.healthcheck,
    w.k8s_namespace,
    COALESCE(e.slug, '') AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON d.id = dt.deployment_id
INNER JOIN `workspaces` w ON w.id = d.workspace_id
INNER JOIN `regions` r ON r.id = dt.region_id
LEFT JOIN `projects` p ON p.id = d.project_id
LEFT JOIN `apps` a ON a.id = d.app_id
LEFT JOIN `environments` e ON e.id = d.environment_id
LEFT JOIN `github_repo_connections` grc ON grc.app_id = d.app_id
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id)
LIMIT 1;
