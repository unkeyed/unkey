-- name: CreateWorkflow :exec
INSERT INTO workflow_executions (
    id,
    workflow_name,
    status,
    input_data,
    max_attempts,
    remaining_attempts,
    namespace,
    trigger_type,
    trigger_source,
    trace_id,
    created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);