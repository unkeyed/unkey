-- name: DeleteSentinelsByEnvironmentId :exec
DELETE FROM sentinels WHERE environment_id = sqlc.arg(environment_id);
