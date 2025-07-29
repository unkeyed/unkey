-- name: UpdateDeploymentOpenapiSpec :exec
UPDATE deployments 
SET openapi_spec = ?, updated_at = ?
WHERE id = ?;