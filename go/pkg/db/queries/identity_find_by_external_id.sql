-- name: FindIdentityByExternalID :one
SELECT id, external_id, workspace_id, environment, created_at, updated_at, meta FROM identities WHERE workspace_id = ? AND external_id = ?
