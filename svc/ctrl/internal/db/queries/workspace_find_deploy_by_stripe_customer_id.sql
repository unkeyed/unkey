-- name: FindDeployWorkspaceByStripeCustomerID :one
-- Resolves a Stripe customer to its Deploy workspace. The ctrl Stripe webhook
-- uses this as the relevance check for month-end invoice closing: invoices of
-- customers without a Deploy plan are left entirely to Stripe's own
-- finalization. The Deploy subscription id now lives on billing_subscriptions.
SELECT
   w.id,
   bs.stripe_subscription_id AS stripe_deploy_subscription_id
FROM `workspace_billing` b
JOIN `workspaces` w ON w.id = b.workspace_id
LEFT JOIN `billing_subscriptions` bs
   ON bs.workspace_id = b.workspace_id AND bs.product = 'compute'
WHERE b.stripe_customer_id = sqlc.arg(stripe_customer_id)
  AND b.plan IS NOT NULL
  AND w.deleted_at_m IS NULL;
