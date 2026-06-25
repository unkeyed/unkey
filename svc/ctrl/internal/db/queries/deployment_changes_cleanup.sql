-- name: DeleteDeploymentChangesBefore :exec
-- DeleteDeploymentChangesBefore removes old deployment_changes entries for TTL-based cleanup.
DELETE FROM `deployment_changes`
WHERE created_at < sqlc.arg(before)
LIMIT 10000;
