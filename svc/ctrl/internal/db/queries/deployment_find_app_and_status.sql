-- name: FindDeploymentAppAndStatus :one
-- FindDeploymentAppAndStatus returns the two columns the create insert checks
-- when it tolerates a row that is already there. Reading the full row for that
-- would carry the encrypted environment variables and sentinel config with it.
SELECT app_id, status
FROM deployments
WHERE id = sqlc.arg('id');
