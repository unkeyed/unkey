-- name: FindInstanceByPodName :one
SELECT
 *
FROM instances
  WHERE k8s_name = sqlc.arg(k8s_name) AND cluster_id = sqlc.arg(cluster_id) AND region = sqlc.arg(region);
