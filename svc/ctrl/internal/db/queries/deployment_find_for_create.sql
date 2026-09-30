-- name: FindDeploymentForCreate :one
-- FindDeploymentForCreate returns the columns Create reads from a row that is
-- already there: the insert checks the app and status, and an approval reuses
-- the private networking decision. Reading the full row would carry the
-- encrypted environment variables and sentinel config with it.
SELECT app_id, status, private_networking
FROM deployments
WHERE id = sqlc.arg('id');
