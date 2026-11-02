-- name: FindPrivateNetworkPreviewDeploymentId :one
-- FindPrivateNetworkPreviewDeploymentId returns the newest deployment that was
-- ever ready on a git branch and fork in a preview environment of the app.
-- Without the hint MySQL may drive from the app's preview environments
-- through app_environment_ready_created_idx and sort the whole branch.
SELECT candidate.id
FROM deployments candidate FORCE INDEX (app_branch_fork_ready_created_idx)
INNER JOIN environments candidate_env ON candidate_env.id = candidate.environment_id
    AND candidate_env.app_id = candidate.app_id
WHERE candidate.app_id = sqlc.arg(app_id)
    AND candidate.git_branch = sqlc.arg(git_branch)
    AND candidate.fork_key = sqlc.arg(fork_key)
    AND candidate.ever_ready = TRUE
    AND candidate.workspace_id = sqlc.arg(workspace_id)
    AND candidate.project_id = sqlc.arg(project_id)
    AND candidate.source = 'git'
    AND candidate_env.kind = 'preview'
ORDER BY candidate.created_at DESC, candidate.id DESC
LIMIT 1;
