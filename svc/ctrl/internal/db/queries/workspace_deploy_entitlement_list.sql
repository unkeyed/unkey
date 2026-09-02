-- name: ListWorkspaceDeployEntitlements :many
-- Batched form of FindWorkspaceDeployEntitlement, for a caller holding several
-- workspaces at once: one GitHub push can match apps across workspaces, and
-- looking each up separately would cost a Restate journal round trip per app.
-- The LEFT JOIN keeps the single-row form's semantics, so a workspace with no
-- billing row comes back with NULL plan columns rather than being dropped, and
-- an unbilled workspace reads the same either way.
SELECT
   w.id AS workspace_id,
   b.plan,
   b.plan_override,
   b.spend_suspended
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON b.workspace_id = w.id
WHERE w.id IN (sqlc.slice('workspace_ids'));
