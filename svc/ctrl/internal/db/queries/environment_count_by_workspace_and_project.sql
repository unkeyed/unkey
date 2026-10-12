-- name: CountEnvironmentsByWorkspaceAndProject :one
-- CountEnvironmentsByWorkspaceAndProject counts a project's environments within a workspace. Only tests use this query.
SELECT COUNT(*)
FROM `environments`
WHERE workspace_id = sqlc.arg(workspace_id) AND project_id = sqlc.arg(project_id);
