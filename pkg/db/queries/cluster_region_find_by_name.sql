-- name: FindRegionByNameAndPlatform :one
SELECT * FROM regions WHERE name = sqlc.arg(name) AND platform = sqlc.arg(platform);
