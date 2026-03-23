-- name: UpdateDeploymentRestateInvocationID :exec
UPDATE deployments
SET restate_invocation_id = ?, updated_at = ?
WHERE id = ?;
