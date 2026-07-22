-- name: FilterExistingDeploymentImages :many
-- FilterExistingDeploymentImages returns image references still used by any
-- deployment. Registry reconciliation preserves these tags even when the
-- deployment ID encoded in the tag no longer exists, as rebuilds reuse images.
SELECT DISTINCT image
FROM deployments
WHERE image IN (sqlc.slice('images'));
