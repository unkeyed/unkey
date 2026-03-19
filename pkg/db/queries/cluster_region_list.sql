-- name: ListRegions :many
SELECT id, name, platform, is_schedulable FROM regions;
