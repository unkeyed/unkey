-- name: RecordInstanceWaiting :exec
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
