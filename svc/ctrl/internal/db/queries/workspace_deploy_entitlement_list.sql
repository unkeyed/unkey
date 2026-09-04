-- name: ListWorkspaceDeployEntitlements :many
-- Batched FindWorkspaceDeployEntitlement, for a push that matches apps across
-- workspaces. The LEFT JOIN keeps a workspace with no billing row, with NULL
-- plan columns.
SELECT
   w.id AS workspace_id,
   b.plan,
   b.plan_override,
   b.spend_suspended
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON b.workspace_id = w.id
WHERE w.id IN (sqlc.slice('workspace_ids'));
