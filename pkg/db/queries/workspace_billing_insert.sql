-- name: InsertWorkspaceBilling :exec
-- Creates the billing row for a workspace, mirroring how UpsertQuota creates the
-- quota row. Idempotent: a second call for the same workspace is a no-op, so it
-- is safe to call after InsertWorkspace/UpsertWorkspace without a prior check.
-- New workspaces start on the Free tier with no Stripe linkage and no plan.
INSERT INTO `workspace_billing` (
    workspace_id,
    tier,
    created_at_m
) VALUES (
    sqlc.arg(workspace_id),
    'Free',
    sqlc.arg(created_at)
)
ON DUPLICATE KEY UPDATE workspace_id = workspace_id;
