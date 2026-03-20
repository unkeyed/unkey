-- name: UpsertRegion :exec
-- Inserts a region or does nothing if it already exists.
INSERT INTO regions (
	id,
	name,
	platform
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(name),
	sqlc.arg(platform)
)
ON DUPLICATE KEY UPDATE name = name;
