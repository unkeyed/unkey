-- name: GetDeploymentChangesMaxVersion :one
-- GetDeploymentChangesMaxVersion returns the current maximum version (pk) for a region.
-- Used during full sync to establish the starting version for incremental polling.
SELECT CAST(COALESCE(MAX(pk), 0) AS UNSIGNED) AS max_version
FROM `deployment_changes`
WHERE region_id = sqlc.arg(region_id);
