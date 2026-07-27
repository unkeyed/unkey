-- name: FindDeploymentTopologyByDeploymentAndRegion :one
-- FindDeploymentTopologyByDeploymentAndRegion returns a single deployment topology with all
-- joined data needed for the Watch stream. Used by the unified WatchDeploymentChanges RPC.
SELECT
    dt.pk AS topology_pk,
    dt.autoscaling_replicas_min AS topology_autoscaling_replicas_min,
    dt.autoscaling_replicas_max AS topology_autoscaling_replicas_max,
    dt.autoscaling_threshold_cpu AS topology_autoscaling_threshold_cpu,
    dt.autoscaling_threshold_memory AS topology_autoscaling_threshold_memory,
    dt.desired_status AS topology_desired_status,
    d.id AS deployment_id,
    d.k8s_name AS deployment_k8s_name,
    d.workspace_id AS deployment_workspace_id,
    d.project_id AS deployment_project_id,
    d.environment_id AS deployment_environment_id,
    d.app_id AS deployment_app_id,
    d.image AS deployment_image,
    d.build_id AS deployment_build_id,
    d.git_commit_sha AS deployment_git_commit_sha,
    d.git_branch AS deployment_git_branch,
    d.git_commit_message AS deployment_git_commit_message,
    d.cpu_millicores AS deployment_cpu_millicores,
    d.memory_mib AS deployment_memory_mib,
    d.storage_mib AS deployment_storage_mib,
    d.encrypted_environment_variables AS deployment_encrypted_environment_variables,
    d.command AS deployment_command,
    d.port AS deployment_port,
    d.shutdown_signal AS deployment_shutdown_signal,
    d.healthcheck AS deployment_healthcheck,
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
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id)
LIMIT 1;
