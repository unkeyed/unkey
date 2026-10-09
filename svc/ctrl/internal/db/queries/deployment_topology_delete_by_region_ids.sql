-- name: DeleteDeploymentTopologiesByRegionIds :execrows
-- DeleteDeploymentTopologiesByRegionIds hard deletes up to limit topology rows
-- in the given regions and reports how many it removed. Only tests use this
-- query.
DELETE FROM `deployment_topology`
WHERE region_id IN (sqlc.slice(region_ids))
LIMIT ?;
