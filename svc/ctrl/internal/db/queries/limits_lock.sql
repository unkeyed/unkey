-- name: LockLimitsByWorkspaceID :one
-- Must be the first statement of its transaction: the usage reads that follow
-- rely on the read view opening after this lock is held
SELECT cpu_cores_max, memory_mib_max, storage_mib_max, custom_domains_max
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id')
FOR UPDATE;
