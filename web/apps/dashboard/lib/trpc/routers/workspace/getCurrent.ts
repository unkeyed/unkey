import { db } from "@/lib/db";
import { subscriptionIdsByProduct } from "@/lib/stripe/billingSubscriptions";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getCurrentWorkspace = protectedProcedure.query(async ({ ctx }) => {
  // createContext already resolved the workspace (with quotas) for this
  // request, so the common case costs no extra query.
  if (ctx.workspace) {
    return ctx.workspace;
  }

  if (!ctx.tenant?.id) {
    // The session has no organization yet (fresh sign-up before onboarding)
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "No organization found - workspace setup required",
    });
  }

  // ctx.workspace is also unset when the context query failed (context
  // creation swallows database errors), so give the lookup one direct
  // attempt before reporting the workspace as missing.
  const orgId = ctx.tenant.id;
  let workspace: Awaited<
    ReturnType<
      typeof db.query.workspaces.findFirst<{
        with: { quotas: true; billing: true; billingSubscriptions: true };
      }>
    >
  >;
  try {
    workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      with: {
        quotas: true,
        billing: true,
        billingSubscriptions: true,
      },
    });
  } catch (error) {
    console.warn("Database error fetching workspace:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch workspace data",
      cause: error,
    });
  }

  if (!workspace) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Workspace not found for organization - workspace setup required",
    });
  }

  // Billing state moved to the workspace_billing relation. Surface it under the
  // legacy workspace field names so existing consumers (the workspace provider,
  // billing pages) read the fresh values from the billing row.
  return {
    ...workspace,
    tier: workspace.billing?.tier ?? "Free",
    stripeCustomerId: workspace.billing?.stripeCustomerId ?? null,
    ...subscriptionIdsByProduct(workspace.billingSubscriptions ?? []),
    deployPlan: workspace.billing?.plan ?? null,
    deployPlanOverride: workspace.billing?.planOverride ?? null,
    deploySpendBudgetCents: workspace.billing?.spendBudgetCents ?? null,
    deploySpendBudgetStop: workspace.billing?.spendBudgetStop ?? false,
    deploySpendSuspended: workspace.billing?.spendSuspended ?? false,
  };
});
