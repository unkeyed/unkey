-- name: ListRegions :many
SELECT id, name, platform, can_schedule FROM regions;
