-- name: StripeConnectedAccountInsert :exec
INSERT INTO stripe_connected_accounts (
  id,
  workspace_id,
  stripe_account_id,
  access_token_encrypted,
  refresh_token_encrypted,
  scope,
  connected_at,
  created_at_m
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: StripeConnectedAccountFindByWorkspaceId :one
SELECT * FROM stripe_connected_accounts
WHERE workspace_id = ? AND disconnected_at IS NULL
LIMIT 1;

-- name: StripeConnectedAccountListActive :many
SELECT * FROM stripe_connected_accounts
WHERE disconnected_at IS NULL AND deleted_at_m IS NULL
ORDER BY created_at_m DESC;

-- name: StripeConnectedAccountDisconnect :exec
UPDATE stripe_connected_accounts
SET disconnected_at = ?, updated_at_m = ?
WHERE workspace_id = ?;

-- name: PricingModelInsert :exec
INSERT INTO pricing_models (
  id,
  workspace_id,
  name,
  currency,
  verification_unit_price,
  key_access_unit_price,
  credit_unit_price,
  tiered_pricing,
  version,
  active,
  created_at_m
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: PricingModelFindById :one
SELECT * FROM pricing_models
WHERE id = ? AND deleted_at_m IS NULL
LIMIT 1;

-- name: PricingModelListByWorkspaceId :many
SELECT * FROM pricing_models
WHERE workspace_id = ? AND deleted_at_m IS NULL
ORDER BY created_at_m DESC;

-- name: PricingModelUpdate :exec
UPDATE pricing_models
SET 
  name = ?,
  verification_unit_price = ?,
  key_access_unit_price = ?,
  credit_unit_price = ?,
  tiered_pricing = ?,
  version = ?,
  updated_at_m = ?
WHERE id = ?;

-- name: PricingModelSoftDelete :exec
UPDATE pricing_models
SET deleted_at_m = ?, active = false, updated_at_m = ?
WHERE id = ?;

-- name: PricingModelCountEndUsers :one
SELECT COUNT(*) as count FROM billing_end_users
WHERE pricing_model_id = ?;

-- name: PricingModelFindWorkspaceCurrency :one
SELECT currency FROM pricing_models
WHERE workspace_id = ? AND deleted_at_m IS NULL
LIMIT 1;

-- name: BillingEndUserInsert :exec
INSERT INTO billing_end_users (
  id,
  workspace_id,
  external_id,
  pricing_model_id,
  stripe_customer_id,
  stripe_subscription_id,
  email,
  name,
  metadata,
  created_at_m
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: BillingEndUserFindById :one
SELECT * FROM billing_end_users
WHERE id = ? AND deleted_at_m IS NULL
LIMIT 1;

-- name: BillingEndUserFindByExternalId :one
SELECT * FROM billing_end_users
WHERE workspace_id = ? AND external_id = ? AND deleted_at_m IS NULL
LIMIT 1;

-- name: BillingEndUserListByWorkspaceId :many
SELECT * FROM billing_end_users
WHERE workspace_id = ? AND deleted_at_m IS NULL
ORDER BY created_at_m DESC;

-- name: BillingEndUserUpdate :exec
UPDATE billing_end_users
SET 
  pricing_model_id = ?,
  stripe_subscription_id = ?,
  email = ?,
  name = ?,
  metadata = ?,
  updated_at_m = ?
WHERE id = ?;

-- name: BillingInvoiceInsert :exec
INSERT INTO billing_invoices (
  id,
  workspace_id,
  end_user_id,
  stripe_invoice_id,
  billing_period_start,
  billing_period_end,
  verification_count,
  key_access_count,
  credits_used,
  total_amount,
  currency,
  status,
  created_at_m
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: BillingInvoiceFindById :one
SELECT * FROM billing_invoices
WHERE id = ?
LIMIT 1;

-- name: BillingInvoiceFindByStripeInvoiceId :one
SELECT * FROM billing_invoices
WHERE stripe_invoice_id = ?
LIMIT 1;

-- name: BillingInvoiceListByWorkspaceId :many
SELECT * FROM billing_invoices
WHERE workspace_id = ?
ORDER BY created_at_m DESC
LIMIT ? OFFSET ?;

-- name: BillingInvoiceListByEndUserId :many
SELECT * FROM billing_invoices
WHERE end_user_id = ?
ORDER BY created_at_m DESC;

-- name: BillingInvoiceUpdateStatus :exec
UPDATE billing_invoices
SET status = ?, updated_at_m = ?
WHERE id = ?;

-- name: BillingTransactionInsert :exec
INSERT INTO billing_transactions (
  id,
  invoice_id,
  stripe_payment_intent_id,
  amount,
  currency,
  status,
  failure_reason,
  created_at_m
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: BillingTransactionListByInvoiceId :many
SELECT * FROM billing_transactions
WHERE invoice_id = ?
ORDER BY created_at_m DESC;

-- name: BillingTransactionUpdateStatus :exec
UPDATE billing_transactions
SET status = ?, failure_reason = ?
WHERE id = ?;
