import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

/**
 * Reports the workspace's Stripe Connect onboarding state for end-user
 * billing: "none" (never started), "pending" (onboarding started, not
 * completed), or "linked" (billable). The account reference itself stays
 * encrypted at rest and is not returned.
 */
export const getStripeConnect = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const settings = await db.query.workspaceBillingSettings.findFirst({
      where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
      columns: { stripeConnectEncrypted: true, stripeConnectStatus: true },
    });

    if (!settings?.stripeConnectEncrypted || !settings.stripeConnectStatus) {
      return { status: "none" as const };
    }
    return { status: settings.stripeConnectStatus };
  });
