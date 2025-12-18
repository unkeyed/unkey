-- name: UpdateSentinelReplicasAndHealth :exec
UPDATE sentinels SET
replicas = sqlc.arg(replicas),
health = sqlc.arg(health),
updated_at = sqlc.arg(updated_at)
WHERE k8s_name = sqlc.arg(k8s_name);
