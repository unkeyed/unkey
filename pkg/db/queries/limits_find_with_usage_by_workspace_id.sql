-- name: FindLimitsWithUsageByWorkspaceID :one
-- The allocation sum matches svc/ctrl reserveTopologies without its
-- exclude_deployment_id filter. Each subquery is covered by a workspace_id index
SELECT
  l.api_billable_operations_count_max_per_month,
  l.api_requests_count_max_per_minute,
  l.logs_retention_days_max,
  l.logs_audit_retention_days_max,
  l.logdrains_max,
  l.cpu_cores_max,
  l.cpu_cores_max_per_instance,
  l.memory_mib_max,
  l.memory_mib_max_per_instance,
  l.storage_mib_max,
  l.storage_mib_max_per_instance,
  l.builds_concurrent_max,
  l.custom_domains_max,
  l.autoscaling_replicas_max,
  b.plan,
  b.plan_override,
  CAST((SELECT COUNT(*) FROM `logdrains` ld WHERE ld.`workspace_id` = l.`workspace_id`) AS SIGNED) AS `logdrains_count`,
  CAST((SELECT COUNT(*) FROM `custom_domains` cd WHERE cd.`workspace_id` = l.`workspace_id`) AS SIGNED) AS `custom_domains_count`,
  CAST(COALESCE(a.`total_cpu_millicores`, 0) AS SIGNED) AS `total_cpu_millicores`,
  CAST(COALESCE(a.`total_memory_mib`, 0) AS SIGNED) AS `total_memory_mib`,
  CAST(COALESCE(a.`total_storage_mib`, 0) AS SIGNED) AS `total_storage_mib`
FROM `limits` l
LEFT JOIN `workspace_billing` b ON b.`workspace_id` = l.`workspace_id`
LEFT JOIN (
  SELECT
    dt.`workspace_id`,
    SUM(d.`cpu_millicores` * dt.`autoscaling_replicas_max`) AS `total_cpu_millicores`,
    SUM(d.`memory_mib` * dt.`autoscaling_replicas_max`) AS `total_memory_mib`,
    SUM(d.`storage_mib` * dt.`autoscaling_replicas_max`) AS `total_storage_mib`
  FROM `deployment_topology` dt
  JOIN `deployments` d ON d.`id` = dt.`deployment_id`
  WHERE dt.`workspace_id` = sqlc.arg('workspace_id')
    AND dt.`desired_status` = 'running'
  GROUP BY dt.`workspace_id`
) a ON a.`workspace_id` = l.`workspace_id`
WHERE l.`workspace_id` = sqlc.arg('workspace_id');
