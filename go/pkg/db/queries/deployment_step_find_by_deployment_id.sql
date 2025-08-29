-- name: FindDeploymentStepsByDeploymentId :many
SELECT
    deployment_id,
    status,
    message,
    created_at
FROM deployment_steps
WHERE deployment_id = ?
ORDER BY created_at ASC;
