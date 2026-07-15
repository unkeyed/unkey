-- name: UpsertCluster :exec
-- Upserts a cluster by region_id. Heartbeats refresh metadata and the timestamp but preserve scheduling state.
INSERT INTO clusters (
	id,
	region_id,
	platform,
	region,
	state,
	last_heartbeat_at
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(region_id),
	sqlc.arg(platform),
	sqlc.arg(region),
	sqlc.arg(state),
	sqlc.arg(last_heartbeat_at)
)
ON DUPLICATE KEY UPDATE
	platform = VALUES(platform),
	region = VALUES(region),
	last_heartbeat_at = VALUES(last_heartbeat_at);
