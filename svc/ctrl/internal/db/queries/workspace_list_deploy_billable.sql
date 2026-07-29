-- name: ListDeployBillableWorkspaces :many
-- Lists every workspace with an active Deploy plan and a Stripe customer:
-- the set whose draft renewal invoices the month-end close finalizes. A
-- workspace that cancelled Deploy mid-month is intentionally absent: cancel
-- clears deploy_plan, so it drops out here and the close never touches it. Its
-- final invoice is left to Stripe's own auto-finalize at the period boundary,
-- which bills whatever the hourly usage push last reported. The push is
-- usage-driven (not gated on deploy_plan), so it keeps reporting until the
-- workloads actually drain, including any usage after the cancel call. The
-- tradeoff is up to ~1h of staleness on that final invoice versus the last
-- hourly tick, the same bound the hourly push accepts everywhere else.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   w.id,
   b.stripe_customer_id,
   bs.stripe_subscription_id AS stripe_deploy_subscription_id
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON (b.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND b.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
LEFT JOIN `billing_subscriptions` bs
   ON (bs.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND bs.workspace_id = w.id COLLATE utf8mb4_0900_as_cs) AND bs.product = 'compute'
WHERE b.plan IS NOT NULL
  AND b.stripe_customer_id IS NOT NULL
  AND w.deleted_at_m IS NULL;
