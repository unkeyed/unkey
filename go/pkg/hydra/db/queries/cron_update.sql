-- name: UpdateCronJobLastRun :exec
UPDATE cron_jobs 
SET last_run_at = ?,
    next_run_at = ?,
    updated_at = ?
WHERE id = ? AND namespace = ?;