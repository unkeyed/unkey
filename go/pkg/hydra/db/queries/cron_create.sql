-- name: CreateCronJob :exec
INSERT INTO cron_jobs (
    id,
    name,
    cron_spec,
    namespace,
    workflow_name,
    enabled,
    created_at,
    updated_at,
    next_run_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateCronJob :exec
UPDATE cron_jobs 
SET cron_spec = ?,
    workflow_name = ?,
    enabled = ?,
    updated_at = ?,
    next_run_at = ?
WHERE id = ?;