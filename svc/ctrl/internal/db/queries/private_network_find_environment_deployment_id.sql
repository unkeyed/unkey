-- name: FindPrivateNetworkEnvironmentDeploymentId :one
-- FindPrivateNetworkEnvironmentDeploymentId returns the newest deployment that
-- was ever ready in a preview environment of the app.
SELECT candidate.id
FROM deployments candidate FORCE INDEX (app_environment_ready_created_idx)
INNER JOIN environments target_env ON target_env.id = candidate.environment_id
    AND target_env.app_id = candidate.app_id
WHERE candidate.app_id = sqlc.arg(app_id)
    AND candidate.environment_id = sqlc.arg(environment_id)
    AND (candidate.first_ready_at IS NOT NULL) = TRUE
    AND candidate.workspace_id = sqlc.arg(workspace_id)
    AND candidate.project_id = sqlc.arg(project_id)
    AND target_env.kind = 'preview'
ORDER BY candidate.created_at DESC, candidate.id DESC
LIMIT 1;
