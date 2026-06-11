-- name: ListDeployBillableWorkspaces :many
-- Lists every workspace with an active Deploy plan and a Stripe customer:
-- the set whose draft renewal invoices the month-end close finalizes. A
-- workspace that cancelled Deploy mid-month is intentionally absent: its
-- final usage was invoiced immediately at cancellation (cancelDeploy uses
-- invoice_now), so there is nothing left for the close to do.
SELECT
   w.id,
   w.stripe_customer_id
FROM `workspaces` w
WHERE w.deploy_plan IS NOT NULL
  AND w.stripe_customer_id IS NOT NULL
  AND w.enabled = true
  AND w.deleted_at_m IS NULL;
