-- name: ListAppConnectionTargetApps :many
SELECT id, name, slug FROM apps
WHERE workspace_id = sqlc.arg(workspace_id) AND project_id = sqlc.arg(project_id)
    AND id <> sqlc.arg(app_id)
ORDER BY name ASC
LIMIT 100;
