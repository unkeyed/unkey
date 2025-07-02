-- name: ResetOrphanedWorkflows :exec
UPDATE workflow_executions we
SET status = 'pending' 
WHERE we.namespace = ? 
  AND we.status = 'running' 
  AND we.id NOT IN (
    SELECT l.resource_id 
    FROM leases l
    WHERE l.kind = 'workflow' AND l.namespace = ?
  );