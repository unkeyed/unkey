-- name: FindAgentSignupByRegistrationID :one
SELECT
    pk,
    id,
    agent_registration_id,
    workos_user_id,
    workspace_id,
    root_key_id,
    status,
    requested_permissions,
    created_at_m,
    updated_at_m
FROM `agent_signups`
WHERE agent_registration_id = sqlc.arg('agent_registration_id');
