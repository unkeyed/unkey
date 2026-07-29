-- name: FindWorkspaceDeployEntitlement :one
-- Reads the Unkey Deploy entitlement signals for the project- and
-- deployment-creation gates: deploy_plan (mirrored from Stripe by the
-- dashboard webhook), deploy_plan_override (manual comp for internal
-- workspaces), and deploy_spend_suspended (the spend cap stopped this
-- workspace's compute). The gates treat either plan column being set as
-- entitled; deployment creation additionally refuses while suspended. Read by
-- ctrl-api outside the billing hot path, so a single lookup by id is fine.
-- Explicit columns (not SELECT *) so the read is insensitive to workspace
-- column ordering.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   b.plan,
   b.plan_override,
   b.spend_suspended
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON (b.workspace_id = w.id COLLATE utf8mb4_0900_ai_ci AND b.workspace_id = w.id COLLATE utf8mb4_0900_as_cs)
WHERE w.id = sqlc.arg(id);
