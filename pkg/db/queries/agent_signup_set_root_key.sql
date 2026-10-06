-- name: SetAgentSignupRootKey :execrows
UPDATE `agent_signups`
SET
    root_key_id = sqlc.arg('root_key_id'),
    status = 'root_key_issued',
    requested_permissions = CAST(sqlc.arg('requested_permissions') AS JSON),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE agent_registration_id = sqlc.arg('agent_registration_id')
  AND status = 'workspace_created'
  AND root_key_id IS NULL;
