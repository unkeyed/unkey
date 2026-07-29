-- name: FindWorkspaceDeployEntitlement :one
-- Reads the Unkey Deploy entitlement signals for the project-creation gate:
-- deploy_plan (mirrored from Stripe by the dashboard webhook) and
-- deploy_plan_override (manual comp for internal workspaces). The gate treats
-- either being set as entitled. Read by ctrl-api outside the billing hot path,
-- so a single lookup by id is fine. Explicit columns (not SELECT *) so the read
-- is insensitive to workspace column ordering.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   b.plan,
   b.plan_override
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON (w.id COLLATE utf8mb4_0900_ai_ci = b.workspace_id AND w.id COLLATE utf8mb4_0900_as_cs = b.workspace_id)
WHERE w.id = sqlc.arg(id);
