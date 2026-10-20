-- name: UpdateWorkspaceBillingTier :exec
UPDATE `workspace_billing`
SET tier = sqlc.arg('tier'), tier_updated_at_m = sqlc.arg('updated_at'), updated_at_m = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id');
