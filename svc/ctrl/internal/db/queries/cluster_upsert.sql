-- name: UpsertCluster :exec
-- UpsertCluster inserts a cluster or refreshes its cell ID and heartbeat.
INSERT INTO clusters (
	id,
	cell_id,
	region_id,
	last_heartbeat_at
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(cell_id),
	sqlc.arg(region_id),
	sqlc.arg(last_heartbeat_at)
)
ON DUPLICATE KEY UPDATE
	cell_id = VALUES(cell_id),
	last_heartbeat_at = VALUES(last_heartbeat_at);
