-- name: ListAppConnectionPinnedDeployments :many
SELECT id, app_id, environment_id, status, git_branch, image_resolved, created_at
FROM deployments
WHERE workspace_id = sqlc.arg(workspace_id) AND project_id = sqlc.arg(project_id)
    AND id IN (sqlc.slice('ids')) AND first_ready_at IS NOT NULL;
