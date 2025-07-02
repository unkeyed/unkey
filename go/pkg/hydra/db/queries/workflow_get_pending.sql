-- name: GetPendingWorkflows :many
SELECT * FROM workflow_executions
WHERE namespace = ? 
  AND (
    status = 'pending' 
    OR (status = 'failed' AND next_retry_at <= ?) 
    OR (status = 'sleeping' AND sleep_until <= ?)
  )
ORDER BY created_at ASC
LIMIT ? OFFSET ?;

-- name: GetPendingWorkflowsWithNames :many
SELECT * FROM workflow_executions
WHERE namespace = ? 
  AND (
    status = 'pending' 
    OR (status = 'failed' AND next_retry_at <= ?) 
    OR (status = 'sleeping' AND sleep_until <= ?)
  )
  AND workflow_name IN (sqlc.slice('workflow_names'))
ORDER BY created_at ASC
LIMIT ? OFFSET ?;