-- name: LockLimitsByWorkspaceID :one
-- Must be the first statement of its transaction: the quota sum that follows
-- relies on the read view opening after this lock is held
SELECT cpu_cores_max, memory_mib_max, storage_mib_max
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id')
FOR UPDATE;
