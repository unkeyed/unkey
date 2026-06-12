-- name: FindDeployWorkspaceByStripeCustomerID :one
-- Resolves a Stripe customer to its workspace, but only when that workspace
-- has an active Deploy plan. The ctrl Stripe webhook uses this as the
-- relevance check for month-end invoice closing: invoices of customers
-- without a Deploy plan are left entirely to Stripe's own finalization.
SELECT
   w.id
FROM `workspaces` w
WHERE w.stripe_customer_id = sqlc.arg(stripe_customer_id)
  AND w.deploy_plan IS NOT NULL
  AND w.deleted_at_m IS NULL;
