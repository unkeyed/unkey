-- name: InsertBillingSubscription :exec
-- InsertBillingSubscription records a workspace's Stripe subscription for one
-- product. Only tests use this query.
INSERT INTO `billing_subscriptions` (
    workspace_id,
    product,
    stripe_subscription_id
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(product),
    sqlc.arg(stripe_subscription_id)
);
