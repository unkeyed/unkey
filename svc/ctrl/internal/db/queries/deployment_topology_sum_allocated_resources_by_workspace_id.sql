-- name: SumAllocatedResourcesByWorkspaceID :one
-- Legacy failure can leave running intent after compute has stopped.
SELECT
  CAST(COALESCE(SUM(d.`cpu_millicores` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_cpu_millicores`,
  CAST(COALESCE(SUM(d.`memory_mib` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_memory_mib`,
  CAST(COALESCE(SUM(d.`storage_mib` * dt.`autoscaling_replicas_max`), 0) AS SIGNED) AS `total_storage_mib`
FROM `deployment_topology` dt
JOIN `deployments` d ON d.`id` = dt.`deployment_id`
WHERE dt.`workspace_id` = sqlc.arg('workspace_id')
  AND dt.`deployment_id` != sqlc.arg('exclude_deployment_id')
  AND d.`desired_state` = 'running'
  AND d.`status` NOT IN ('failed', 'cancelled', 'superseded', 'skipped');
