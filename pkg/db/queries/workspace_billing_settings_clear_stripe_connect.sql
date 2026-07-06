-- name: ClearWorkspaceBillingStripeConnect :exec
-- Unlink: delete the encrypted reference so the period-close push skips the
-- workspace. Recorded period rate cards and rollups are unaffected.
UPDATE workspace_billing_settings
SET stripe_connect_encrypted = NULL,
    stripe_connect_encryption_key_id = NULL,
    stripe_connect_status = NULL
WHERE workspace_id = sqlc.arg(workspace_id);
