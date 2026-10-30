import { db } from "../db";
import { subscriptionIdsByProduct } from "../stripe/billingSubscriptions";

export async function loadWorkspace(orgId: string) {
  const workspace = await db.query.workspaces.findFirst({
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
    where: (workspace, { and, eq, isNull }) =>
      and(eq(workspace.orgId, orgId), isNull(workspace.deletedAtM)),
    with: {
      limits: {
        columns: {
          pk: true,
          workspaceId: true,
          apiBillableOperationsCountMaxPerMonth: true,
          apiRequestsCountMaxPerMinute: true,
          logsRetentionDaysMax: true,
          logsAuditRetentionDaysMax: true,
          logdrainsMax: true,
          teamEnabled: true,
          rootKeyIpAllowlistEnabled: true,
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
  });
  if (!workspace) {
    return undefined;
  }

  const { billing, billingSubscriptions } = workspace;
  return {
    ...workspace,
    tier: billing?.tier ?? "Free",
    stripeCustomerId: billing?.stripeCustomerId ?? null,
    ...subscriptionIdsByProduct(billingSubscriptions),
    deployPlan: billing?.plan ?? null,
    deployPlanOverride: billing?.planOverride ?? null,
    deploySpendBudgetCents: billing?.spendBudgetCents ?? null,
    deploySpendBudgetStop: billing?.spendBudgetStop ?? false,
    deploySpendSuspended: billing?.spendSuspended ?? false,
  };
}
