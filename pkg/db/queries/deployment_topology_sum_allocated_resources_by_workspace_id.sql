-- name: SumAllocatedResourcesByWorkspaceID :one
-- The sum svc/ctrl reserveTopologies checks against the workspace limits,
-- without its exclude_deployment_id filter. Covered by workspace_idx on
-- deployment_topology
SELECT
  CAST(COALESCE(SUM(d.`cpu_millicores` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_cpu_millicores`,
  CAST(COALESCE(SUM(d.`memory_mib` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_memory_mib`,
  CAST(COALESCE(SUM(d.`storage_mib` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_storage_mib`
FROM `deployment_topology` dt
JOIN `deployments` d ON d.`id` = dt.`deployment_id`
WHERE dt.`workspace_id` = sqlc.arg('workspace_id')
  AND dt.`desired_status` = 'running';
