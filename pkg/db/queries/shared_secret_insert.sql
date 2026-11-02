-- name: InsertSharedSecret :exec
-- InsertSharedSecret creates a fixture with caller-controlled expiry and
-- encryption metadata, allowing tests to cover retained and rotated secrets.
INSERT INTO shared_secrets (
    id,
    workspace_id,
    expires_at,
    created_at,
    encrypted,
    encryption_key_id
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(expires_at),
    sqlc.arg(created_at),
    sqlc.arg(encrypted),
    sqlc.arg(encryption_key_id)
);
