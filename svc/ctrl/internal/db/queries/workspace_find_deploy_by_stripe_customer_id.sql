-- name: FindDeployWorkspaceByStripeCustomerID :one
-- Resolves a Stripe customer to its Deploy workspace. The ctrl Stripe webhook
-- uses this as the relevance check for month-end invoice closing: invoices of
-- customers without a Deploy plan are left entirely to Stripe's own
-- finalization.
SELECT
   w.id,
   b.stripe_subscription_id
FROM `workspace_billing` b
JOIN `workspaces` w ON w.id = b.workspace_id
WHERE b.stripe_customer_id = sqlc.arg(stripe_customer_id)
  AND b.plan IS NOT NULL
  AND w.deleted_at_m IS NULL;
