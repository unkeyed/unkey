-- name: ListWorkspacesForQuotaCheck :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   w.id,
   w.org_id,
   w.name,
   b.stripe_customer_id,
   b.tier,
   w.enabled,
   q.requests_per_month
FROM `workspaces` w
LEFT JOIN quota q ON (q.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND q.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
LEFT JOIN `workspace_billing` b ON (b.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND b.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
WHERE w.id > sqlc.arg('cursor')
ORDER BY w.id ASC
LIMIT 100;

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
LEFT JOIN quota q ON (q.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND q.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
LEFT JOIN `workspace_billing` b ON (b.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND b.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
WHERE w.id IN (sqlc.slice('workspace_ids'));
