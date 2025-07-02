-- name: CreateStep :exec
INSERT INTO workflow_steps (
    id,
    execution_id,
    step_name,
    step_order,
    status,
    max_attempts,
    remaining_attempts,
    namespace
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);