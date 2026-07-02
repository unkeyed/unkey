-- name: ListEnvironmentIdsByApp :many
SELECT id FROM environments WHERE app_id = sqlc.arg(app_id);
