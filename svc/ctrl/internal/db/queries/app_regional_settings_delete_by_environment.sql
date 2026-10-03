-- name: DeleteAppRegionalSettingsByEnvironmentId :exec
DELETE s, p
FROM app_regional_settings s
LEFT JOIN app_regional_settings other
    ON other.horizontal_autoscaling_policy_id = s.horizontal_autoscaling_policy_id
    AND other.environment_id <> s.environment_id
LEFT JOIN horizontal_autoscaling_policies p
    ON p.id = s.horizontal_autoscaling_policy_id AND other.pk IS NULL
WHERE s.environment_id = sqlc.arg(environment_id);
