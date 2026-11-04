-- name: InsertCliSession :exec
-- Inserts a CLI session. Used only by tests; the API owns session creation.
INSERT INTO cli_sessions (id, workspace_id, user_id, user_name, permissions, refresh_token_hash, refresh_family, created_at, last_used_at, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(user_id), sqlc.arg(user_name), sqlc.arg(permissions), sqlc.arg(refresh_token_hash), sqlc.arg(refresh_family), sqlc.arg(created_at), sqlc.arg(created_at), sqlc.arg(expires_at));
