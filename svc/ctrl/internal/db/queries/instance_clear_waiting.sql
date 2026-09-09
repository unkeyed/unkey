-- name: ClearInstanceWaiting :exec
-- A running event is authoritative for the current container life. Remove a
-- stale startup/pod error once kubelet has successfully started the same or a
-- newer restart, while watch-observation and restartCount guards protect
-- against delayed events from an older snapshot.
UPDATE instances
SET container_status = JSON_SET(
	JSON_REMOVE(container_status, '$.waiting'),
	'$.restartCount', CAST(sqlc.arg(restart_count) AS UNSIGNED),
	'$.statusObservedAt', CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND COALESCE(CAST(JSON_VALUE(container_status, '$.statusObservedAt') AS UNSIGNED), 0) <= CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
	AND CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) <= CAST(sqlc.arg(restart_count) AS UNSIGNED);
