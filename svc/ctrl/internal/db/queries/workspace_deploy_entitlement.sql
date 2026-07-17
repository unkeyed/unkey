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
SELECT
   w.deploy_plan,
   w.deploy_plan_override,
   w.deploy_spend_suspended
FROM `workspaces` w
WHERE w.id = sqlc.arg(id);
