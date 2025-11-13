-- name: FindInstancesByDeploymentId :many
SELECT * FROM instance WHERE deployment_id = ?;