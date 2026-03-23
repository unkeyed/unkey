-- name: FindAppRegionalSettingsByAppAndEnv :many
SELECT
	ars.region_id,
	r.name AS region_name,
	ars.replicas,
	r.can_schedule AS region_can_schedule
FROM app_regional_settings ars
JOIN regions r ON r.id = ars.region_id
WHERE ars.app_id = sqlc.arg(app_id)
  AND ars.environment_id = sqlc.arg(environment_id);
