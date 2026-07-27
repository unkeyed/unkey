-- name: DeleteWorkspaceBillingSubscriptions :exec
-- Removes every Stripe subscription row for a workspace. Paired with
-- ResetWorkspaceBilling by the `unkey dev stripe reset` tooling.
DELETE FROM `billing_subscriptions` WHERE workspace_id = sqlc.arg(id);
