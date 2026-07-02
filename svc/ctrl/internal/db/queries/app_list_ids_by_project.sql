-- name: ListAppIdsByProject :many
SELECT id FROM apps WHERE project_id = sqlc.arg(project_id);
