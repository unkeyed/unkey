-- name: RecordDeploymentPodFailure :exec
-- Retains terminal pod diagnostics after instance cleanup. Recovery does not
-- clear history. Repeated and older observations cannot replace a newer one.
UPDATE deployments
SET last_pod_failure = JSON_OBJECT(
  'podUid', CAST(sqlc.arg(pod_uid) AS CHAR),
  'podName', CAST(sqlc.arg(pod_name) AS CHAR),
  'regionId', CAST(sqlc.arg(region_id) AS CHAR),
  'reason', CAST(sqlc.arg(reason) AS CHAR),
  'message', CAST(sqlc.arg(message) AS CHAR),
  'observedAt', CAST(sqlc.arg(observed_at) AS UNSIGNED)
)
WHERE id = sqlc.arg(deployment_id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND COALESCE(CAST(JSON_VALUE(last_pod_failure, '$.observedAt') AS UNSIGNED), 0) < CAST(sqlc.arg(observed_at) AS UNSIGNED);
