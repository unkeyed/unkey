-- name: ReconcileInstanceContainerStatus :exec
UPDATE instances
SET container_status = JSON_SET(
	CASE WHEN CAST(sqlc.arg(waiting_reason) AS CHAR) = ''
		THEN JSON_REMOVE(container_status, '$.waiting')
		ELSE JSON_SET(container_status, '$.waiting', JSON_OBJECT(
			'reason', CAST(sqlc.arg(waiting_reason) AS CHAR),
			'message', CAST(sqlc.arg(waiting_message) AS CHAR)
		))
	END,
	'$.restartCount', CAST(sqlc.arg(restart_count) AS UNSIGNED),
	'$.statusObservedAt', CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND COALESCE(CAST(JSON_VALUE(container_status, '$.statusObservedAt') AS UNSIGNED), 0) <= CAST(sqlc.arg(status_observed_at) AS UNSIGNED);
