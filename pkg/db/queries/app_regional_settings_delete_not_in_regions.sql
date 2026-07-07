-- name: DeleteAppRegionalSettingsNotInRegions :exec
-- Deletes an environment's regional rows whose region is not in the desired set,
-- reconciling the stored set to exactly the provided regions in one statement.
DELETE FROM app_regional_settings
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND region_id NOT IN (sqlc.slice(region_ids));
