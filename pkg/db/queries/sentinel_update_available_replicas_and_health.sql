-- name: UpdateSentinelAvailableReplicasAndHealth :exec
UPDATE sentinels SET
available_replicas = sqlc.arg(available_replicas),
health = sqlc.arg(health),
updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(sentinel_id);
