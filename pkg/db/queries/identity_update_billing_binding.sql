-- name: UpdateIdentityBillingBinding :exec
UPDATE identities
SET billing_provider = sqlc.arg(billing_provider),
    billing_external_customer_id = sqlc.arg(billing_external_customer_id)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(identity_id)
  AND deleted = false;
