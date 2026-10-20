-- name: FindDeploymentForCreate :one
SELECT app_id, status, capabilities, encrypted_environment_variables
FROM deployments
WHERE id = sqlc.arg('id');
