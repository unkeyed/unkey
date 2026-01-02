-- name: UpdateSentinelAvailableReplicasAndHealth :exec
UPDATE sentinels SET
available_replicas = sqlc.arg(available_replicas),
health = sqlc.arg(health),
updated_at = sqlc.arg(updated_at)
WHERE k8s_name = sqlc.arg(k8s_name);
