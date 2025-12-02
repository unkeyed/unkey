

-- name: UpdateInstanceStatus :exec
UPDATE instances SET
	status = sqlc.arg(status)
WHERE pod_name = sqlc.arg(pod_name) AND shard = sqlc.arg(shard) AND region = sqlc.arg(region);
