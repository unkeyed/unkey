-- name: DeleteEnvironmentsByAppId :exec
DELETE FROM environments WHERE app_id = sqlc.arg(app_id);
