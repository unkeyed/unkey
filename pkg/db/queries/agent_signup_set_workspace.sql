-- name: SetAgentSignupWorkspace :execrows
UPDATE `agent_signups`
SET
    workspace_id = sqlc.arg('workspace_id'),
    status = sqlc.arg('status'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE agent_registration_id = sqlc.arg('agent_registration_id')
  AND status = 'pending';
