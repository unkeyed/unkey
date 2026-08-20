-- name: UpsertWorkspaceBillingSpendSuspended :exec
INSERT INTO `workspace_billing` (
    workspace_id,
    spend_suspended,
    created_at_m
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(spend_suspended),
    sqlc.arg(created_at_m)
)
ON DUPLICATE KEY UPDATE
    spend_suspended = VALUES(spend_suspended);
