-- name: GetWorkflow :one
SELECT * FROM workflow_executions 
WHERE id = ? AND namespace = ?;

-- name: CreateWorkflow :exec
INSERT INTO workflow_executions (
    id, workflow_name, status, input_data, output_data, error_message,
    created_at, started_at, completed_at, max_attempts, remaining_attempts,
    next_retry_at, namespace, trigger_type, trigger_source, sleep_until,
    trace_id, span_id
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?
);

-- name: GetPendingWorkflows :many
SELECT * FROM workflow_executions 
WHERE namespace = ? 
  AND (
    status = 'pending' 
    OR (status = 'failed' AND next_retry_at <= ?) 
    OR (status = 'sleeping' AND sleep_until <= ?)
  )
ORDER BY created_at ASC 
LIMIT ?;

-- name: GetPendingWorkflowsFiltered :many
SELECT * FROM workflow_executions 
WHERE namespace = ? 
  AND (
    status = 'pending' 
    OR (status = 'failed' AND next_retry_at <= ?) 
    OR (status = 'sleeping' AND sleep_until <= ?)
  )
  AND workflow_name IN (/*SLICE:workflow_names*/?)
ORDER BY created_at ASC 
LIMIT ?;

-- name: UpdateWorkflowFields :exec
UPDATE workflow_executions 
SET 
  status = COALESCE(?, status),
  error_message = COALESCE(?, error_message),
  completed_at = COALESCE(?, completed_at),
  started_at = COALESCE(?, started_at),
  output_data = COALESCE(?, output_data),
  remaining_attempts = COALESCE(?, remaining_attempts),
  next_retry_at = COALESCE(?, next_retry_at),
  sleep_until = COALESCE(?, sleep_until)
WHERE id = ? AND workflow_executions.namespace = ?
  AND EXISTS (
    SELECT 1 FROM leases 
    WHERE resource_id = ? AND kind = 'workflow' 
    AND worker_id = ? AND expires_at > ?
  );

-- name: UpdateStepStatus :exec
UPDATE workflow_steps 
SET status = ?, completed_at = ?, output_data = ?, error_message = ?
WHERE namespace = ? AND execution_id = ? AND step_name = ?;

-- name: SleepWorkflow :exec
UPDATE workflow_executions 
SET status = 'sleeping', sleep_until = ?
WHERE id = ? AND namespace = ?;

-- name: CreateStep :exec
INSERT INTO workflow_steps (
    id, execution_id, step_name, status, output_data, error_message,
    started_at, completed_at, max_attempts, remaining_attempts, namespace
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?
);

-- name: GetStep :one
SELECT * FROM workflow_steps 
WHERE namespace = ? AND execution_id = ? AND step_name = ?;

-- name: GetCompletedStep :one
SELECT * FROM workflow_steps 
WHERE namespace = ? AND execution_id = ? AND step_name = ? AND status = 'completed';

-- name: UpdateStepStatusWithLease :exec
UPDATE workflow_steps 
SET status = ?, completed_at = ?, output_data = ?, error_message = ?
WHERE workflow_steps.namespace = ? AND execution_id = ? AND step_name = ?
  AND EXISTS (
    SELECT 1 FROM leases 
    WHERE resource_id = ? AND kind = 'workflow' 
    AND worker_id = ? AND expires_at > ?
  );

-- name: GetLease :one
SELECT * FROM leases 
WHERE resource_id = ? AND kind = ?;

-- name: CreateLease :exec
INSERT INTO leases (
    resource_id, kind, namespace, worker_id, acquired_at, expires_at, heartbeat_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateLease :exec
UPDATE leases 
SET worker_id = ?, acquired_at = ?, expires_at = ?, heartbeat_at = ?
WHERE resource_id = ? AND kind = ? AND expires_at < ?;

-- name: UpdateWorkflowToRunning :exec
UPDATE workflow_executions 
SET status = 'running', 
    started_at = CASE WHEN started_at IS NULL THEN ? ELSE started_at END,
    sleep_until = NULL
WHERE id = ? AND workflow_executions.namespace = ?
  AND EXISTS (
    SELECT 1 FROM leases 
    WHERE resource_id = ? AND kind = 'workflow' 
    AND worker_id = ? AND expires_at > ?
  );

-- name: CompleteWorkflow :exec
UPDATE workflow_executions 
SET status = 'completed', completed_at = ?, output_data = ?
WHERE id = ? AND namespace = ?;

-- name: HeartbeatLease :exec
UPDATE leases 
SET heartbeat_at = ?, expires_at = ?
WHERE resource_id = ? AND worker_id = ?;

-- name: ReleaseLease :exec
DELETE FROM leases 
WHERE resource_id = ? AND worker_id = ?;

-- name: GetSleepingWorkflows :many
SELECT * FROM workflow_executions 
WHERE namespace = ? AND status = 'sleeping' AND sleep_until <= ?
ORDER BY sleep_until ASC;

-- name: GetCronJob :one
SELECT * FROM cron_jobs 
WHERE namespace = ? AND name = ?;

-- name: GetCronJobs :many
SELECT * FROM cron_jobs 
WHERE namespace = ? AND enabled = true;

-- name: GetDueCronJobs :many
SELECT * FROM cron_jobs 
WHERE namespace = ? AND enabled = true AND next_run_at <= ?;

-- name: CreateCronJob :exec
INSERT INTO cron_jobs (
    id, name, cron_spec, namespace, workflow_name, enabled, 
    created_at, updated_at, last_run_at, next_run_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON DUPLICATE KEY UPDATE 
    cron_spec = sqlc.arg('cron_spec'), enabled = sqlc.arg('enabled'), updated_at = sqlc.arg('updated_at'), next_run_at = sqlc.arg('next_run_at'), last_run_at = sqlc.arg('last_run_at'), next_run_at = sqlc.arg('next_run_at');

-- name: UpdateCronJob :exec
UPDATE cron_jobs 
SET cron_spec = ?, workflow_name = ?, enabled = ?, updated_at = ?, next_run_at = ?
WHERE id = ? AND namespace = ?;

-- name: UpdateCronJobLastRun :exec
UPDATE cron_jobs 
SET last_run_at = ?, next_run_at = ?, updated_at = ?
WHERE id = ? AND namespace = ?;

-- name: CleanupExpiredLeases :exec
DELETE FROM leases 
WHERE namespace = ? AND expires_at < ?;


-- name: ResetOrphanedWorkflows :exec
UPDATE workflow_executions 
SET status = 'pending' 
WHERE workflow_executions.namespace = ? 
  AND workflow_executions.status = 'running' 
  AND workflow_executions.id NOT IN (
    SELECT resource_id 
    FROM leases 
    WHERE kind = 'workflow' AND leases.namespace = ?
  );

