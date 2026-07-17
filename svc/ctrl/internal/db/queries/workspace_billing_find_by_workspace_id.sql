-- name: FindWorkspaceBillingByWorkspaceID :one
-- Reads a workspace's billing row directly (Stripe linkage, tier, Compute plan,
-- spend budget and spend-cap state). Used by the Deploy cancel path to read the
-- current plan and Stripe subscription. When a workspace is already being
-- fetched, prefer joining workspace_billing in that query over a second round
-- trip.
SELECT * FROM `workspace_billing`
WHERE workspace_id = sqlc.arg(workspace_id);
