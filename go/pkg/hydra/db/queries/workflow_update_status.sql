-- name: UpdateWorkflowStatus :exec
UPDATE workflow_executions 
SET status = ?, error_message = ?
WHERE id = ? AND namespace = ?;

-- name: UpdateWorkflowStatusRunning :exec
UPDATE workflow_executions 
SET status = 'running',
    started_at = CASE WHEN started_at IS NULL THEN ? ELSE started_at END,
    sleep_until = NULL
WHERE id = ? AND namespace = ?;