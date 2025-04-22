-- name: FindIdentityByID :one
SELECT id, external_id, workspace_id, environment, created_at, updated_at, meta, deleted FROM identities WHERE id = sqlc.arg(id) AND deleted = ?;
