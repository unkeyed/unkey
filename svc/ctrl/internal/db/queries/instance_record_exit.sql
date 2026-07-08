-- name: RecordInstanceExit :exec
-- Denormalizes the most recent container exit info onto the instances row's
-- container_status JSON. Called by ctrl when krane reports an
-- event_kind='terminated' event.
--
-- The caller computes the full new ContainerStatus value (restartCount,
-- lastTerminationState, no waiting) and passes it in one typed param. The
-- WHERE clause inspects the row's *existing* container_status to drop
-- delayed events; once the guard passes, the new value fully replaces the
-- old (including clearing $.waiting, since a fresh exit ends any prior
-- crashloop window).
--
-- Out-of-order events from krane are dropped via a lexicographic
-- (restartCount, finishedAt) tuple comparison: an incoming row only wins
-- if its (restartCount, finishedAt) pair is strictly greater than the
-- pair already on the row. The previous OR-of-clauses formulation let a
-- delayed terminated event from restart_count-1 sneak past via the
-- finishedAt branch and regress the row after restart_count had already
-- advanced.
UPDATE instances
SET container_status = sqlc.arg(container_status)
WHERE k8s_name = sqlc.arg(k8s_name)
	AND region_id = sqlc.arg(region_id)
	AND (
		CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) < CAST(sqlc.arg(restart_count) AS UNSIGNED)
		OR (
			CAST(JSON_VALUE(container_status, '$.restartCount') AS UNSIGNED) = CAST(sqlc.arg(restart_count) AS UNSIGNED)
			AND (
				JSON_VALUE(container_status, '$.lastTerminationState.finishedAt') IS NULL
				OR CAST(JSON_VALUE(container_status, '$.lastTerminationState.finishedAt') AS UNSIGNED) < CAST(sqlc.arg(finished_at) AS UNSIGNED)
			)
		)
	);
