-- name: ListRunningDeploymentsByBranch :many
-- ListRunningDeploymentsByBranch returns deployments in the same app,
-- environment, and branch whose desired state is running, excluding one
-- deployment id, the app's current deployment, and deployments from another
-- source fork. Pinned deployments are included.
SELECT d.id
FROM deployments d
WHERE d.git_branch <=> sqlc.arg(git_branch)
  AND d.workspace_id = sqlc.arg(workspace_id)
  AND d.project_id = sqlc.arg(project_id)
  AND d.app_id = sqlc.arg(app_id)
  AND d.environment_id = sqlc.arg(environment_id)
  AND d.fork_repository_full_name <=> (
      SELECT newer.fork_repository_full_name FROM deployments newer WHERE newer.id = sqlc.arg(not_deployment_id)
  )
  AND d.desired_state = 'running'
  AND d.id != sqlc.arg(not_deployment_id)
  AND NOT EXISTS (
      SELECT 1 FROM apps a WHERE a.id = d.app_id AND a.current_deployment_id = d.id
  )
ORDER BY d.created_at ASC;
