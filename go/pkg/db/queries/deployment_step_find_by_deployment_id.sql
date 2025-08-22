-- name: FindDeploymentStepsByDeploymentId :many
SELECT 
    deployment_id,
    status,
    message,
    error_message,
    created_at
FROM deployment_steps 
WHERE deployment_id = ?
ORDER BY created_at ASC;