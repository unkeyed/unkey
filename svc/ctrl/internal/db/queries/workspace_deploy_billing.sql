-- name: ListWorkspacesForDeployBillingByIDs :many
-- Fetches the Stripe customer identity for a batch of workspaces, used by the
-- hourly Deploy billing push to decide where each workspace's month-to-date
-- usage gets reported. The Stripe Billing Meters map usage to a customer by
-- stripe_customer_id, so that (not a subscription or price) is all the push
-- needs. Batched by ID (never per-workspace) so the push stays a single round
-- trip regardless of how many workspaces had usage.
SELECT
   w.id,
   b.stripe_customer_id,
   w.enabled
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON b.workspace_id = w.id
WHERE w.id IN (sqlc.slice('workspace_ids'));

-- name: ListDeployBillingCustomers :many
-- Lists the workspaces whose Deploy usage can be reported to Stripe. This is
-- intentionally not gated on an active plan or enabled workspace: usage
-- incurred while a cancelled deployment drains is still owed. The hourly
-- push uses this set to scope and shard the ClickHouse scan before doing the
-- expensive checkpoint integration; workspaces without a Stripe customer
-- could never produce a meter event and must not make that scan more costly.
SELECT
   w.id,
   b.stripe_customer_id
FROM `workspaces` w
INNER JOIN `workspace_billing` b ON b.workspace_id = w.id
WHERE b.stripe_customer_id IS NOT NULL
  AND b.stripe_customer_id <> '';
