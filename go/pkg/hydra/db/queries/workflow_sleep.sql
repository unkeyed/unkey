-- name: SleepWorkflow :exec
UPDATE workflow_executions 
SET status = 'sleeping',
    sleep_until = ?
WHERE id = ? AND namespace = ?;

-- name: GetSleepingWorkflows :many
SELECT * FROM workflow_executions
WHERE namespace = ? 
  AND status = 'sleeping' 
  AND sleep_until <= ?
ORDER BY sleep_until ASC;