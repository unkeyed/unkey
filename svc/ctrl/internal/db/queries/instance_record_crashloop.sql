-- name: RecordInstanceCrashLoopBackOff :exec
-- Records that kubelet has put a container into CrashLoopBackOff by setting
-- container_status.waiting.reason. The lastTerminationState carries the
-- most recent exit info and is left untouched — the dashboard renders both
-- the underlying exit and the "currently throttling" indicator together.
--
-- Called once per (pod_uid, container_name, restart_count) when krane sees
-- the waiting container reach the BackOff state. The next terminated event
-- (or a successful start) will remove $.waiting via RecordInstanceExit.
--
-- Out-of-order events are dropped via the restartCount guard: a delayed
-- crashloop RPC from an earlier container life cannot flip the waiting
-- reason back after RecordInstanceExit has already advanced restartCount
-- and removed $.waiting.
UPDATE instances
SET container_status = JSON_SET(
	container_status,
	'$.waiting', JSON_OBJECT('reason', 'CrashLoopBackOff')
)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) <= CAST(sqlc.arg(restart_count) AS UNSIGNED);
