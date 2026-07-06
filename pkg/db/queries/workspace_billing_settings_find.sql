-- name: FindWorkspaceBillingSettings :one
SELECT *
FROM workspace_billing_settings
WHERE workspace_id = sqlc.arg(workspace_id);
