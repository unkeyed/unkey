-- name: SetWorkspaceBillingStripeConnect :exec
INSERT INTO workspace_billing_settings (
    id,
    workspace_id,
    stripe_connect_encrypted,
    stripe_connect_encryption_key_id,
    stripe_connect_status,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(stripe_connect_encrypted),
    sqlc.arg(stripe_connect_encryption_key_id),
    sqlc.arg(stripe_connect_status),
    sqlc.arg(created_at)
) ON DUPLICATE KEY UPDATE
    stripe_connect_encrypted = VALUES(stripe_connect_encrypted),
    stripe_connect_encryption_key_id = VALUES(stripe_connect_encryption_key_id),
    stripe_connect_status = VALUES(stripe_connect_status),
    updated_at = VALUES(created_at);
