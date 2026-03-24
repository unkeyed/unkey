-- name: FindInstancesByDeploymentID :many
-- FindInstancesByDeploymentID returns all instances for a given deployment.
-- Used by the router to determine which regions have running instances
-- for instance-aware routing decisions.
SELECT *
FROM instances
WHERE deployment_id = sqlc.arg(deployment_id);
