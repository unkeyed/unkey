-- name: UpsertRegion :exec
-- Inserts a region with the provided can_schedule value, or does nothing if
-- the region already exists. This preserves any manual can_schedule override
-- made directly in the database (e.g. disabling a broken region).
INSERT INTO regions (
	id,
	name,
	platform,
	can_schedule
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(name),
	sqlc.arg(platform),
	sqlc.arg(can_schedule)
)
ON DUPLICATE KEY UPDATE name = name;
