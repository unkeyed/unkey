-- name: ListRunningDeploymentsByBranch :many
-- ListRunningDeploymentsByBranch returns deployments in the same app,
-- environment, and branch whose desired state is running, excluding one
-- deployment id. Used to find sibling running deployments without including
-- the caller's own deployment or unrelated deployments from another scope.
SELECT id
FROM deployments
WHERE git_branch = sqlc.arg(git_branch)
  AND workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND desired_state = 'running'
  AND id != sqlc.arg(not_deployment_id)
ORDER BY created_at ASC;
