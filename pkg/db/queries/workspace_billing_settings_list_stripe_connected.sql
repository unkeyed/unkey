-- name: ListStripeConnectedWorkspaces :many
-- Only fully-linked workspaces are billed: accounts mid-onboarding
-- (status "pending") are excluded until Stripe reports details_submitted.
SELECT *
FROM workspace_billing_settings
WHERE stripe_connect_encrypted IS NOT NULL
  AND stripe_connect_status = 'linked'
ORDER BY pk
LIMIT ? OFFSET ?;
