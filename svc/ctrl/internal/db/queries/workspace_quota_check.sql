-- name: GetWorkspacesForQuotaCheckByIDs :many
SELECT
   w.id,
   w.org_id,
   w.name,
   b.stripe_customer_id,
   b.tier,
   w.enabled,
   q.requests_per_month
FROM `workspaces` w
LEFT JOIN quota q ON q.workspace_id = w.id
LEFT JOIN `workspace_billing` b ON b.workspace_id = w.id
WHERE w.id IN (sqlc.slice('workspace_ids'));
