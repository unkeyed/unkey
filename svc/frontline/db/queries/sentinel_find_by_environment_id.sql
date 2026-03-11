-- name: FindSentinelsByEnvironmentID :many
SELECT sqlc.embed(s), sqlc.embed(r) FROM sentinels s LEFT JOIN regions r ON s.region_id = r.id WHERE s.environment_id = sqlc.arg(environment_id);
