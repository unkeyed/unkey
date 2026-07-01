-- name: ListAppRegionalSettingsByAppEnv :many
-- Returns the current regional rows for reconciliation, including the
-- horizontal_autoscaling_policy_id that FindAppRegionalSettingsByAppAndEnv omits.
SELECT
    region_id,
    replicas,
    horizontal_autoscaling_policy_id
FROM app_regional_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
