-- Complex queries for workflow lease acquisition transaction

-- name: GetWorkflowForLease :one
SELECT * FROM workflow_executions 
WHERE id = ? AND namespace = ?;

-- name: GetExistingLease :one
SELECT * FROM leases 
WHERE resource_id = ? AND kind = 'workflow';

-- name: CheckLeaseExpired :one
SELECT EXISTS(
    SELECT 1 FROM leases 
    WHERE resource_id = ? AND expires_at > ?
) as lease_active;