-- name: ListWorkspacesForDeployBillingByIDs :many
-- Fetches the Stripe customer identity for a batch of workspaces, used by the
-- hourly Deploy billing push to decide where each workspace's month-to-date
-- usage gets reported. The Stripe Billing Meters map usage to a customer by
-- stripe_customer_id, so that (not a subscription or price) is all the push
-- needs. Batched by ID (never per-workspace) so the push stays a single round
-- trip regardless of how many workspaces had usage.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   w.id,
   b.stripe_customer_id,
   w.enabled
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON (w.id COLLATE utf8mb4_0900_ai_ci = b.workspace_id AND w.id COLLATE utf8mb4_0900_as_cs = b.workspace_id)
WHERE w.id IN (sqlc.slice('workspace_ids'));
