-- name: ResetWorkspaceBilling :exec
-- Clears the workspace_billing linkage on a workspace, returning it to the
-- Free tier. Mirrors what the customer.subscription.deleted webhook writes,
-- plus stripe_customer_id, which no webhook ever clears. Stripe subscription
-- ids live on billing_subscriptions and are cleared separately by
-- DeleteWorkspaceBillingSubscriptions. Used by the `unkey dev stripe reset`
-- tooling.
UPDATE `workspace_billing`
SET stripe_customer_id = NULL,
    plan = NULL,
    tier = 'Free'
WHERE workspace_id = sqlc.arg(id);
