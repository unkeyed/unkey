-- name: RecordInstanceExit :exec
UPDATE instances
SET container_status = JSON_SET(
	CASE
		WHEN COALESCE(CAST(JSON_VALUE(container_status, '$.statusObservedAt') AS UNSIGNED), 0) <= CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
			AND CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) <= CAST(sqlc.arg(restart_count) AS UNSIGNED)
		THEN JSON_SET(
			JSON_REMOVE(container_status, '$.waiting'),
			'$.restartCount', CAST(sqlc.arg(restart_count) AS UNSIGNED),
			'$.statusObservedAt', CAST(sqlc.arg(status_observed_at) AS UNSIGNED)
		)
		ELSE container_status
	END,
	'$.lastTerminationState', JSON_OBJECT(
		'exitCode', CAST(sqlc.arg(exit_code) AS SIGNED),
		'signal', CAST(sqlc.arg(signal) AS SIGNED),
		'reason', CAST(sqlc.arg(reason) AS CHAR),
		'finishedAt', CAST(sqlc.arg(finished_at) AS UNSIGNED)
	)
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND COALESCE(CAST(JSON_VALUE(container_status, '$.lastTerminationState.finishedAt') AS UNSIGNED), 0) < CAST(sqlc.arg(finished_at) AS UNSIGNED);
