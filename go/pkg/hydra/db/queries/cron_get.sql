-- name: GetCronJob :one
SELECT * FROM cron_jobs 
WHERE namespace = ? AND name = ?;

-- name: GetCronJobs :many
SELECT * FROM cron_jobs 
WHERE namespace = ? AND enabled = true;

-- name: GetDueCronJobs :many
SELECT * FROM cron_jobs 
WHERE namespace = ? AND enabled = true AND next_run_at <= ?;