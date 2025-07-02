-- name: GetAllWorkflows :many
SELECT * FROM workflow_executions 
WHERE namespace = ?;

-- name: GetAllSteps :many
SELECT * FROM workflow_steps 
WHERE namespace = ?;