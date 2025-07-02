-- name: FailWorkflow :exec
UPDATE workflow_executions 
SET status = 'failed',
    error_message = ?,
    remaining_attempts = ?,
    completed_at = ?,
    next_retry_at = ?
WHERE id = ? AND namespace = ?;

-- name: FailWorkflowFinal :exec
UPDATE workflow_executions 
SET status = 'failed',
    error_message = ?,
    remaining_attempts = ?,
    completed_at = ?,
    next_retry_at = NULL
WHERE id = ? AND namespace = ?;