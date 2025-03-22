-- name: FindRatelimitsByIdentityID :many
SELECT id, name, workspace_id, created_at, updated_at, `limit`, duration FROM ratelimits WHERE identity_id = sqlc.arg(identity_id)
