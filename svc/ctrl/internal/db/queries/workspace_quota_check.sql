-- name: ListWorkspacesForQuotaCheck :many
SELECT
   w.id,
   w.org_id,
   w.name,
   w.stripe_customer_id,
   w.tier,
   w.enabled,
   q.requests_per_month
FROM `workspaces` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id > sqlc.arg('cursor')
ORDER BY w.id ASC
LIMIT 100;

-- name: GetWorkspacesForQuotaCheckByIDs :many
SELECT
   w.id,
   w.org_id,
   w.name,
   w.stripe_customer_id,
   w.tier,
   w.enabled,
   q.requests_per_month
FROM `workspaces` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id IN (sqlc.slice('workspace_ids'));
