-- name: CompareAndSwapDeploymentStatus :execresult
UPDATE deployments
SET status = sqlc.arg(new_status), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
AND status = sqlc.arg(expected_status);
