-- name: ListAppConnectionTargetEnvironments :many
SELECT id, app_id, slug, kind FROM environments
WHERE workspace_id = sqlc.arg(workspace_id) AND project_id = sqlc.arg(project_id)
ORDER BY slug ASC
LIMIT 500;
