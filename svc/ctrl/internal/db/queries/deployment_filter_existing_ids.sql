-- name: FilterExistingDeploymentIds :many
-- FilterExistingDeploymentIds returns the subset of the given deployment ids
-- that exist. Used by the registry sweep to decide which Depot image tags
-- are orphaned.
SELECT id FROM deployments
WHERE id IN (sqlc.slice('ids'));
