-- name: FindAppRegionalSettingsByAppAndEnv :many
-- FindAppRegionalSettingsByAppAndEnv returns per-region deployment settings
-- including the autoscaling policy values (if attached) for snapshotting
-- into deployment_topology at deploy time. A row may reference either an
-- HPA or VPA policy (never both).
SELECT
	ars.region_id,
	r.name AS region_name,
	ars.replicas,
	r.can_schedule AS region_can_schedule,
	hap.replicas_min AS autoscaling_replicas_min,
	hap.replicas_max AS autoscaling_replicas_max,
	hap.cpu_threshold AS autoscaling_threshold_cpu,
	hap.memory_threshold AS autoscaling_threshold_memory,
	vap.update_mode AS vpa_update_mode,
	vap.controlled_resources AS vpa_controlled_resources,
	vap.controlled_values AS vpa_controlled_values,
	vap.cpu_min_millicores AS vpa_cpu_min_millicores,
	vap.cpu_max_millicores AS vpa_cpu_max_millicores,
	vap.memory_min_mib AS vpa_memory_min_mib,
	vap.memory_max_mib AS vpa_memory_max_mib
FROM app_regional_settings ars
JOIN regions r ON r.id = ars.region_id
LEFT JOIN horizontal_autoscaling_policies hap ON hap.id = ars.horizontal_autoscaling_policy_id
LEFT JOIN vertical_autoscaling_policies vap ON vap.id = ars.vertical_autoscaling_policy_id
WHERE ars.app_id = sqlc.arg(app_id)
  AND ars.environment_id = sqlc.arg(environment_id);
