-- name: ListAppRegionalSettingsByApp :many
-- Returns per-region settings for every environment in an app, including the
-- autoscaling policy bounds (if attached). Callers group by environment_id.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
	ars.environment_id,
	ars.region_id,
	r.name AS region_name,
	ars.replicas,
	r.can_schedule AS region_can_schedule,
	hap.replicas_min AS autoscaling_replicas_min,
	hap.replicas_max AS autoscaling_replicas_max,
	hap.cpu_threshold AS autoscaling_threshold_cpu,
	hap.memory_threshold AS autoscaling_threshold_memory
FROM app_regional_settings ars
JOIN regions r ON (ars.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND ars.region_id COLLATE utf8mb4_0900_as_cs = r.id)
LEFT JOIN horizontal_autoscaling_policies hap ON hap.id = ars.horizontal_autoscaling_policy_id
WHERE ars.app_id = sqlc.arg(app_id);
