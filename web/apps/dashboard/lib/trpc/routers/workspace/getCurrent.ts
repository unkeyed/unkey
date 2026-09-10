import { db } from "@/lib/db";
import { subscriptionIdsByProduct } from "@/lib/stripe/billingSubscriptions";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

const workspaceProjection = {
  columns: {
    pk: true,
    id: true,
    orgId: true,
    name: true,
    slug: true,
    k8sNamespace: true,
    betaFeatures: true,
    subscriptions: true,
    enabled: true,
    deleteProtection: true,
    createdAtM: true,
    updatedAtM: true,
    deletedAtM: true,
  },
  with: {
    limits: {
      columns: {
        pk: true,
        workspaceId: true,
        apiBillableOperationsCountMaxPerMonth: true,
        apiRequestsCountMaxPerMinute: true,
        logsRetentionDaysMax: true,
        logsAuditRetentionDaysMax: true,
        teamEnabled: true,
        cpuCoresMax: true,
        cpuCoresMaxPerInstance: true,
        memoryMibMax: true,
        memoryMibMaxPerInstance: true,
        storageMibMax: true,
        storageMibMaxPerInstance: true,
        buildsConcurrentMax: true,
        customDomainsMax: true,
        autoscalingReplicasMax: true,
      },
    },
    billing: {
      columns: {
        pk: true,
        workspaceId: true,
        tier: true,
        stripeCustomerId: true,
        plan: true,
        planOverride: true,
        spendBudgetCents: true,
        spendBudgetStop: true,
        spendSuspended: true,
        createdAtM: true,
        updatedAtM: true,
        deletedAtM: true,
      },
    },
    billingSubscriptions: {
      columns: {
        pk: true,
        workspaceId: true,
        product: true,
        stripeSubscriptionId: true,
        createdAt: true,
        updatedAt: true,
      },
    },
  },
} satisfies NonNullable<Parameters<typeof db.query.workspaces.findFirst>[0]>;

export const getCurrentWorkspace = protectedProcedure.query(async ({ ctx }) => {
  // createContext already resolved the workspace (with limits) for this
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
    ReturnType<typeof db.query.workspaces.findFirst<typeof workspaceProjection>>
  >;
  try {
    workspace = await db.query.workspaces.findFirst({
      columns: {
        pk: true,
        id: true,
        orgId: true,
        name: true,
        slug: true,
        k8sNamespace: true,
        betaFeatures: true,
        subscriptions: true,
        enabled: true,
        deleteProtection: true,
        createdAtM: true,
        updatedAtM: true,
        deletedAtM: true,
      },
      with: {
        limits: {
          columns: {
            pk: true,
            workspaceId: true,
            apiBillableOperationsCountMaxPerMonth: true,
            apiRequestsCountMaxPerMinute: true,
            logsRetentionDaysMax: true,
            logsAuditRetentionDaysMax: true,
            teamEnabled: true,
            cpuCoresMax: true,
            cpuCoresMaxPerInstance: true,
            memoryMibMax: true,
            memoryMibMaxPerInstance: true,
            storageMibMax: true,
            storageMibMaxPerInstance: true,
            buildsConcurrentMax: true,
            customDomainsMax: true,
            autoscalingReplicasMax: true,
          },
        },
        billing: {
          columns: {
            pk: true,
            workspaceId: true,
            tier: true,
            stripeCustomerId: true,
            plan: true,
            planOverride: true,
            spendBudgetCents: true,
            spendBudgetStop: true,
            spendSuspended: true,
            createdAtM: true,
            updatedAtM: true,
            deletedAtM: true,
          },
        },
        billingSubscriptions: {
          columns: {
            pk: true,
            workspaceId: true,
            product: true,
            stripeSubscriptionId: true,
            createdAt: true,
            updatedAt: true,
          },
        },
      },
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
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
