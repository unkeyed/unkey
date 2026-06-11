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
SELECT
   w.id,
   w.stripe_customer_id,
   w.stripe_subscription_id
FROM `workspaces` w
WHERE w.deploy_plan IS NOT NULL
  AND w.stripe_customer_id IS NOT NULL
  AND w.enabled = true
  AND w.deleted_at_m IS NULL;
