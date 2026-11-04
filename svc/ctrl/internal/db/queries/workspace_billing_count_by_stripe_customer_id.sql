-- name: CountOtherWorkspaceBillingsByStripeCustomerID :one
-- Counts the other workspaces billed to a Stripe customer. A GDPR workspace
-- erasure deletes the customer, so it refuses when another workspace shares it.
SELECT COUNT(*) FROM workspace_billing
WHERE stripe_customer_id = sqlc.arg(stripe_customer_id)
  AND workspace_id != sqlc.arg(workspace_id);
