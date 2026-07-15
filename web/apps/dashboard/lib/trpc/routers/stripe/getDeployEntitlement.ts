import { workspaceProcedure } from "../../trpc";

/**
 * Whether the workspace may create Unkey Deploy projects: it has a synced Deploy
 * plan (deploy_plan) or a manual override (deploy_plan_override). Read from the
 * local signals on the workspace, no Stripe call. This is the dashboard UX
 * mirror of the authoritative Go gate in ctrl-api.
 */
export const getDeployEntitlement = workspaceProcedure.query(({ ctx }) => {
  const entitled = Boolean(ctx.workspace.deployPlan) || Boolean(ctx.workspace.deployPlanOverride);
  return { entitled };
});
