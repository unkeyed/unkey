-- name: RecordInstanceWaiting :exec
-- Records an actionable kubelet waiting or pod-level failure reason. The
-- lastTerminationState carries the most recent process exit and is left
-- untouched so the dashboard can render both pieces of context together.
--
-- Called once per (pod_uid, container_name, restart_count) when krane sees
-- a startup failure or terminal pod reason. The next terminated event or a
-- successful start removes $.waiting.
--
-- Out-of-order events are dropped via the watch-observation and restartCount
-- guards: a delayed waiting RPC cannot flip the reason back after a newer
-- running event removed $.waiting.
UPDATE instances
SET container_status = JSON_SET(
	container_status,
	'$.restartCount', CAST(sqlc.arg(restart_count) AS UNSIGNED),
	'$.statusObservedAt', CAST(sqlc.arg(status_observed_at) AS UNSIGNED),
	'$.waiting', JSON_OBJECT(
		'reason', sqlc.arg(reason),
		'message', sqlc.arg(message)
	)
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND COALESCE(CAST(JSON_VALUE(container_status, '$.statusObservedAt') AS UNSIGNED), 0) <= CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
	AND CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) <= CAST(sqlc.arg(restart_count) AS UNSIGNED);
