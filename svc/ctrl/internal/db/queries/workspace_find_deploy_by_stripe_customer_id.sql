-- name: FindDeployWorkspaceByStripeCustomerID :one
-- Resolves a Stripe customer to its Deploy workspace. The ctrl Stripe webhook
-- uses this as the relevance check for month-end invoice closing: invoices of
-- customers without a Deploy plan are left entirely to Stripe's own
-- finalization. The Deploy subscription id now lives on billing_subscriptions.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   w.id,
   bs.stripe_subscription_id AS stripe_deploy_subscription_id
FROM `workspace_billing` b
JOIN `workspaces` w ON (w.id = b.workspace_id COLLATE utf8mb4_0900_ai_ci AND w.id = b.workspace_id COLLATE utf8mb4_0900_as_cs)
LEFT JOIN `billing_subscriptions` bs
   ON (bs.workspace_id = b.workspace_id COLLATE utf8mb4_0900_ai_ci AND bs.workspace_id = b.workspace_id COLLATE utf8mb4_0900_as_cs) AND bs.product = 'compute'
WHERE b.stripe_customer_id = sqlc.arg(stripe_customer_id)
  AND b.plan IS NOT NULL
  AND w.deleted_at_m IS NULL;
