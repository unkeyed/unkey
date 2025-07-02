-- name: GetStep :one
SELECT * FROM workflow_steps 
WHERE namespace = ? AND execution_id = ? AND step_name = ?;

-- name: GetCompletedStep :one
SELECT * FROM workflow_steps 
WHERE namespace = ? AND execution_id = ? AND step_name = ? AND status = 'completed';