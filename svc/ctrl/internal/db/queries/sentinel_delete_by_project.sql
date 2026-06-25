-- name: DeleteSentinelsByProjectId :exec
DELETE FROM sentinels WHERE project_id = sqlc.arg(project_id);
