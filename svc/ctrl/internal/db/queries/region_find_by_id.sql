-- name: FindRegionById :one
SELECT
 *
FROM regions
WHERE id = sqlc.arg(region_id) LIMIT 1;
