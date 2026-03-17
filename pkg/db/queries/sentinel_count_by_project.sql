-- name: CountSentinelsByProjectId :one
SELECT COUNT(*) as count FROM sentinels WHERE project_id = sqlc.arg(project_id);
