-- name: CompleteWorkflow :exec
UPDATE workflow_executions 
SET status = 'completed',
    completed_at = ?,
    output_data = ?
WHERE id = ? AND namespace = ?;