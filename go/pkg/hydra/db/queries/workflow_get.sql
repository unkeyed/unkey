-- name: GetWorkflow :one
SELECT * FROM workflow_executions 
WHERE id = ? AND namespace = ?;