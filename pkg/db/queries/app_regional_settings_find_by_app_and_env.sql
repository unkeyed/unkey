-- name: FindAppRegionalSettingsByAppAndEnv :many
-- FindAppRegionalSettingsByAppAndEnv returns per-region deployment settings
-- including the autoscaling policy values (if attached) for snapshotting
-- into deployment_topology at deploy time.
SELECT
	ars.region_id,
	r.name AS region_name,
	ars.replicas,
	r.can_schedule AS region_can_schedule,
	hap.replicas_min AS autoscaling_replicas_min,
	hap.replicas_max AS autoscaling_replicas_max,
	hap.cpu_threshold AS autoscaling_threshold_cpu,
	hap.memory_threshold AS autoscaling_threshold_memory
FROM app_regional_settings ars
JOIN regions r ON r.id = ars.region_id
LEFT JOIN horizontal_autoscaling_policies hap ON hap.id = ars.horizontal_autoscaling_policy_id
WHERE ars.app_id = sqlc.arg(app_id)
  AND ars.environment_id = sqlc.arg(environment_id);
