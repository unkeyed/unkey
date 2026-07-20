-- name: ResetWorkspaceBilling :exec
-- Clears every billing linkage on a workspace, returning it to the Free
-- tier. Mirrors what the customer.subscription.deleted webhook writes, plus
-- stripe_customer_id, which no webhook ever clears. Used by the
-- `unkey dev stripe reset` tooling; quota is reset separately via UpdateQuota.
UPDATE `workspace_billing`
SET stripe_customer_id = NULL,
    stripe_subscription_id = NULL,
    plan = NULL,
    tier = 'Free'
WHERE workspace_id = sqlc.arg(id);
