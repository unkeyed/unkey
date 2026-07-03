-- name: UpdateDeploymentInvocationID :exec
UPDATE deployments
SET invocation_id = ?, updated_at = ?
WHERE id = ?;
