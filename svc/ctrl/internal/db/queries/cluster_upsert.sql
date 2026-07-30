-- name: UpsertCluster :exec
-- UpsertCluster inserts a cluster or lets an existing region claim its cell ID
-- exactly once. Conflicting cell or region identities remain unchanged so the
-- caller can detect and reject them after this query.
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
	last_heartbeat_at = IF(
		region_id = VALUES(region_id) AND (cell_id IS NULL OR cell_id = VALUES(cell_id)),
		VALUES(last_heartbeat_at),
		last_heartbeat_at
	),
	cell_id = COALESCE(cell_id, VALUES(cell_id));
