-- name: GetWorkspacesForQuotaCheckByIDs :many
-- Rows for workspaces already over the ClickHouse usage floor.
-- api_requests_count_max_per_minute is the workspace API throttle: NULL means
-- there is no throttle, and any stored value, including zero, is already
-- enforced. api_subscription_id is set only when billing_subscriptions has a
-- row with product 'api'. subscriptions is the legacy custom-pricing blob;
-- NULL or an empty object means the workspace is on the current billing path.
SELECT
   w.id,
   w.org_id,
   w.name,
   w.slug,
   w.enabled,
   w.deleted_at_m,
   w.subscriptions,
   b.stripe_customer_id,
   b.tier,
   l.api_billable_operations_count_max_per_month AS requests_per_month,
   l.api_requests_count_max_per_minute,
   s.stripe_subscription_id AS api_subscription_id
FROM `workspaces` w
LEFT JOIN `limits` l ON l.workspace_id = w.id
LEFT JOIN `workspace_billing` b ON b.workspace_id = w.id
LEFT JOIN `billing_subscriptions` s
   ON s.workspace_id = w.id AND s.product = 'api'
WHERE w.id IN (sqlc.slice('workspace_ids'));
