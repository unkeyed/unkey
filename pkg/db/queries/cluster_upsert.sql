-- name: UpsertCluster :exec
-- Upserts a cluster by region_id. If the cluster already exists, updates the heartbeat timestamp.
INSERT INTO clusters (
	id,
	region_id,
	last_heartbeat_at
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(region_id),
	sqlc.arg(last_heartbeat_at)
)
ON DUPLICATE KEY UPDATE
	last_heartbeat_at = sqlc.arg(last_heartbeat_at);
