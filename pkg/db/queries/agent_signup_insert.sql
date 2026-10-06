-- name: InsertAgentSignup :exec
INSERT INTO `agent_signups` (
    id,
    agent_registration_id,
    workos_user_id,
    workspace_id,
    root_key_id,
    status,
    requested_permissions,
    created_at_m,
    updated_at_m
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('agent_registration_id'),
    sqlc.arg('workos_user_id'),
    sqlc.arg('workspace_id'),
    sqlc.arg('root_key_id'),
    sqlc.arg('status'),
    CAST(sqlc.arg('requested_permissions') AS JSON),
    sqlc.arg('created_at_m'),
    sqlc.arg('updated_at_m')
);
