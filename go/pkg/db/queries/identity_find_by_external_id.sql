-- name: FindIdentityByExternalID :one
SELECT id, external_id, workspace_id, environment, created_at, updated_at, meta, deleted FROM identities WHERE workspace_id = ? AND external_id = ? AND deleted = ?;
