-- name: ListDeploymentChangesByRegionAll :many
-- ListDeploymentChangesByRegionAll returns all deployment changes for a region with version > after_version.
-- Used by the unified WatchDeploymentChanges stream. Does not filter by resource_type.
SELECT deployment_changes.pk, deployment_changes.resource_type, deployment_changes.resource_id, deployment_changes.region_id, deployment_changes.created_at
FROM `deployment_changes`
WHERE pk > sqlc.arg(after_version) AND region_id = sqlc.arg(region_id)
ORDER BY pk ASC
LIMIT ?;
