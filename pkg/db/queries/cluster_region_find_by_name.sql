-- name: FindRegionByNameAndPlatform :one
SELECT id FROM regions WHERE name = sqlc.arg(name) AND platform = sqlc.arg(platform);
