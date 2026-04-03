-- name: FindDeploymentTopologyByIDAndRegion :one
SELECT
    d.id,
    d.k8s_name,
    w.k8s_namespace,
    d.workspace_id,
    d.project_id,
    d.environment_id,
    d.app_id,
    d.build_id,
    d.image,
    r.name AS region,
    d.cpu_millicores,
    d.memory_mib,
    d.storage_mib,
    dt.autoscaling_replicas_min,
    dt.autoscaling_replicas_max,
    dt.autoscaling_threshold_cpu,
    dt.autoscaling_threshold_memory,
    dt.vpa_update_mode,
    dt.vpa_controlled_resources,
    dt.vpa_controlled_values,
    dt.vpa_cpu_min_millicores,
    dt.vpa_cpu_max_millicores,
    dt.vpa_memory_min_mib,
    dt.vpa_memory_max_mib,
    dt.desired_status,
    d.encrypted_environment_variables,
    d.command,
    d.port,
    d.shutdown_signal,
    d.healthcheck,
    d.git_commit_sha,
    d.git_branch,
    d.git_commit_message,
    e.slug AS environment_slug,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
INNER JOIN `regions` r ON dt.region_id = r.id
INNER JOIN `environments` e ON d.environment_id = e.id
LEFT JOIN `github_repo_connections` grc ON d.app_id = grc.app_id
WHERE  r.name = sqlc.arg(region)
    AND dt.deployment_id = sqlc.arg(deployment_id)
LIMIT 1;
