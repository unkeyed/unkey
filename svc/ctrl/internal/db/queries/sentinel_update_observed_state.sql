-- name: UpdateSentinelObservedState :exec
-- UpdateSentinelObservedState writes observed state from a krane agent:
-- the current health, available replica count, and the image that is
-- actually running on the pods. The running image is used to detect
-- rollout convergence — a deploy is only complete when running_image
-- matches the desired image.
UPDATE sentinels SET
  available_replicas = sqlc.arg(available_replicas),
  health = sqlc.arg(health),
  running_image = sqlc.arg(running_image),
  updated_at = sqlc.arg(updated_at)
WHERE k8s_name = sqlc.arg(k8s_name);
