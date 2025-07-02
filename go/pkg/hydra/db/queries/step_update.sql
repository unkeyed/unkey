-- name: UpdateStepStatus :exec
UPDATE workflow_steps 
SET status = ?,
    completed_at = ?,
    output_data = ?,
    error_message = ?
WHERE namespace = ? AND execution_id = ? AND step_name = ?;