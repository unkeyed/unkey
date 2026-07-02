-- name: FindWorkspaceDeployEntitlement :one
-- Reads the Unkey Deploy entitlement signals for the project-creation gate:
-- deploy_plan (mirrored from Stripe by the dashboard webhook) and
-- deploy_plan_override (manual comp for internal workspaces). The gate treats
-- either being set as entitled. Read by ctrl-api outside the billing hot path,
-- so a single lookup by id is fine. Explicit columns (not SELECT *) so the read
-- is insensitive to workspace column ordering.
SELECT
   w.deploy_plan,
   w.deploy_plan_override
FROM `workspaces` w
WHERE w.id = sqlc.arg(id);
