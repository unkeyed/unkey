-- name: FindSentinelsByEnvironmentID :many
SELECT * FROM sentinels WHERE environment_id = sqlc.arg(environment_id);
