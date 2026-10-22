-- name: RecordInstanceRunning :exec
-- Records that a container is running again after a restart.
--
-- The guard is strict. A delayed running event of the life that then crashed
-- has the same restart_count as the row and must not remove that crash loop.
-- After restartCount moves forward, RecordInstanceCrashLoopBackOff rejects
-- delayed crash loop events of earlier lives.
UPDATE instances
SET container_status = JSON_SET(
	JSON_REMOVE(container_status, '$.waiting'),
	'$.restartCount', CAST(sqlc.arg(restart_count) AS UNSIGNED)
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) < CAST(sqlc.arg(restart_count) AS UNSIGNED);
