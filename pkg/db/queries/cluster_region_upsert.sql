-- name: UpsertRegion :exec
-- Inserts a region with the provided is_schedulable value, or does nothing if
-- the region already exists. This preserves any manual is_schedulable override
-- made directly in the database (e.g. disabling a broken region).
INSERT INTO regions (
	id,
	name,
	platform,
	is_schedulable
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(name),
	sqlc.arg(platform),
	sqlc.arg(is_schedulable)
)
ON DUPLICATE KEY UPDATE name = name;
