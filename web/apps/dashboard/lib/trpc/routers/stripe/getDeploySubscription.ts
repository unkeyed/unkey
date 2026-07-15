import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import { workspaceProcedure } from "../../trpc";

/**
 * Returns the workspace's current Unkey Deploy plan, read from the local
 * deploy_plan signal (synced from Stripe by the webhook). No Stripe call on
 * read; null means no Deploy plan.
 */
export const getDeploySubscription = workspaceProcedure.query(({ ctx }) => {
  const raw = ctx.workspace.deployPlan;
  const plan: DeployPlan | null =
    raw && (DEPLOY_PLANS as readonly string[]).includes(raw) ? (raw as DeployPlan) : null;
  return { plan };
});
