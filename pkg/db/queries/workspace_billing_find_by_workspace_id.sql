-- name: FindWorkspaceBillingByWorkspaceID :one
-- Reads a workspace's billing row directly (Stripe linkage, tier, Compute plan,
-- spend budget and spend-cap state). Use this when only billing state is needed;
-- when a workspace is already being fetched, prefer joining workspace_billing in
-- that query rather than a second round trip. Stripe subscription ids now live
-- on billing_subscriptions, one row per (workspace, product).
SELECT
   b.pk,
   b.workspace_id,
   b.tier,
   b.stripe_customer_id,
   bs_api.stripe_subscription_id AS stripe_subscription_id,
   bs_deploy.stripe_subscription_id AS stripe_deploy_subscription_id,
   b.plan,
   b.plan_override,
   b.spend_budget_cents,
   b.spend_budget_stop,
   b.spend_suspended,
   b.created_at_m,
   b.updated_at_m,
   b.deleted_at_m
FROM `workspace_billing` b
LEFT JOIN `billing_subscriptions` bs_api
   ON b.workspace_id = bs_api.workspace_id AND bs_api.product = 'api'
LEFT JOIN `billing_subscriptions` bs_deploy
   ON b.workspace_id = bs_deploy.workspace_id AND bs_deploy.product = 'compute'
WHERE b.workspace_id = sqlc.arg(workspace_id);
