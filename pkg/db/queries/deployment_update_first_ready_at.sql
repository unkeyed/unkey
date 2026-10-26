-- name: UpdateDeploymentFirstReadyAt :exec
-- UpdateDeploymentFirstReadyAt is used by API tests to model reaching ready.
UPDATE deployments
SET first_ready_at = COALESCE(first_ready_at, sqlc.arg(first_ready_at))
WHERE id = sqlc.arg(id);
