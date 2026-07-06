-- name: UpsertWorkspaceBillingSettingsDefaultRateCard :exec
INSERT INTO workspace_billing_settings (
    id,
    workspace_id,
    default_rate_card_id,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(default_rate_card_id),
    sqlc.arg(created_at)
) ON DUPLICATE KEY UPDATE
    default_rate_card_id = VALUES(default_rate_card_id),
    updated_at = VALUES(created_at);
