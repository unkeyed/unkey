-- name: SetWorkspaceStripeCustomerId :exec
-- SetWorkspaceStripeCustomerId links a workspace's billing row to a Stripe
-- customer. Only tests use this query.
UPDATE `workspace_billing`
SET stripe_customer_id = sqlc.arg(stripe_customer_id)
WHERE workspace_id = sqlc.arg(workspace_id);
