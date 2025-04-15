-- name: FindIdentityByID :one
SELECT external_id, workspace_id, environment, meta, created_at, updated_at FROM identities WHERE id = sqlc.arg(id)
