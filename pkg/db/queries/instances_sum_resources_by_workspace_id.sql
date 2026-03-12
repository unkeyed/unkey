-- name: SumAllocatedResourcesByWorkspaceID :one
-- Calculates the total allocated CPU and memory for a workspace by summing
-- (deployment cpu/memory × desired replicas) across all running topologies.
SELECT
  CAST(COALESCE(SUM(d.`cpu_millicores` * dt.`desired_replicas`), 0) AS SIGNED) AS `total_cpu_millicores`,
  CAST(COALESCE(SUM(d.`memory_mib` * dt.`desired_replicas`), 0) AS SIGNED) AS `total_memory_mib`
FROM `deployment_topology` dt
JOIN `deployments` d ON d.`id` = dt.`deployment_id`
WHERE dt.`workspace_id` = sqlc.arg('workspace_id')
  AND dt.`desired_status` = 'running';
