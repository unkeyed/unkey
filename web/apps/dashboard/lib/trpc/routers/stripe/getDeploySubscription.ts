import { logOperation } from "@/lib/logging";
import { drizzleComputeSignupCreditClaimStore } from "@/lib/stripe/computeSignupCreditClaims";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import { workspaceProcedure } from "../../trpc";

/**
 * Returns the workspace's current Unkey Deploy plan, read from the local
 * deploy_plan signal (synced from Stripe by the webhook). No Stripe call on
 * read; null means no Deploy plan. signupCreditClaimed is true when this
 * workspace or the signed-in user already has a claim row.
 */
export const getDeploySubscription = workspaceProcedure.query(async ({ ctx }) => {
  const raw = ctx.workspace.deployPlan;
  const plan: DeployPlan | null =
    raw && (DEPLOY_PLANS as readonly string[]).includes(raw) ? (raw as DeployPlan) : null;
  try {
    const signupCreditClaimed = await drizzleComputeSignupCreditClaimStore.existsForWorkspaceOrUser(
      ctx.workspace.id,
      ctx.user.id,
    );
    return { plan, signupCreditClaimed };
  } catch (error) {
    logOperation("error", "Failed to read compute signup credit claim", {
      workspace_id: ctx.workspace.id,
      error_message: error instanceof Error ? error.message : "unknown",
    });
    return { plan, signupCreditClaimed: false };
  }
});
