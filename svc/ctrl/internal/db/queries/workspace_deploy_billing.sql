-- name: ListWorkspacesForDeployBillingByIDs :many
-- Fetches the Stripe customer identity for a batch of workspaces, used by the
-- hourly Deploy billing push to decide where each workspace's month-to-date
-- usage gets reported. The Stripe Billing Meters map usage to a customer by
-- stripe_customer_id, so that (not a subscription or price) is all the push
-- needs. Batched by ID (never per-workspace) so the push stays a single round
-- trip regardless of how many workspaces had usage.
SELECT
   w.id,
   w.stripe_customer_id,
   w.enabled
FROM `workspaces` w
WHERE w.id IN (sqlc.slice('workspace_ids'));
