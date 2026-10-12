-- name: UpsertRegionWithCanSchedule :exec
-- UpsertRegionWithCanSchedule inserts a region with an explicit can_schedule
-- flag, or overwrites the flag if the region already exists. Intended for
-- tests: production registers regions through UpsertRegion and leaves
-- can_schedule at its default.
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
ON DUPLICATE KEY UPDATE can_schedule = sqlc.arg(can_schedule);
