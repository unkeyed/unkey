-- name: FindWorkspaceBillingByWorkspaceID :one
-- Reads a workspace's billing row directly (Stripe linkage, tier, Compute plan,
-- spend budget and spend-cap state). Use this when only billing state is needed;
-- when a workspace is already being fetched, prefer joining workspace_billing in
-- that query rather than a second round trip.
SELECT * FROM `workspace_billing`
WHERE workspace_id = sqlc.arg(workspace_id);
