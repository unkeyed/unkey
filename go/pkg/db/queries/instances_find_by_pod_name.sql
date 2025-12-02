-- name: FindInstanceByPodName :one
SELECT
 *
FROM instances
WHERE pod_name = sqlc.arg(pod_name) AND shard = sqlc.arg(shard) AND region = sqlc.arg(region);
