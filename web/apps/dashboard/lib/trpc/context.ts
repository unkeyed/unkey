import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

import { getAuth } from "../auth/get-auth";
import { getClientIp } from "../client-ip";
import { db } from "../db";
import { subscriptionIdsByProduct } from "../stripe/billingSubscriptions";

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
    quotas: {
      columns: {
        pk: true,
        workspaceId: true,
        requestsPerMonth: true,
        logsRetentionDays: true,
        auditLogsRetentionDays: true,
        team: true,
        ratelimitApiLimit: true,
        ratelimitApiDuration: true,
        allocatedCpuMillicoresTotal: true,
        allocatedMemoryMibTotal: true,
        allocatedStorageMibTotal: true,
        maxCpuMillicoresPerInstance: true,
        maxMemoryMibPerInstance: true,
        maxStorageMibPerInstance: true,
        maxConcurrentBuilds: true,
        maxReplicasPerRegion: true,
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

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const authResult = await getAuth(req as NextRequest);
  const { userId, orgId } = authResult;

  let ws: Awaited<ReturnType<typeof db.query.workspaces.findFirst<typeof workspaceProjection>>> =
    undefined;

  // Only attempt workspace query if we have both userId and orgId
  // This prevents unnecessary queries during auth setup phase
  if (orgId && userId) {
    try {
      ws = await db.query.workspaces.findFirst({
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
          quotas: {
            columns: {
              pk: true,
              workspaceId: true,
              requestsPerMonth: true,
              logsRetentionDays: true,
              auditLogsRetentionDays: true,
              team: true,
              ratelimitApiLimit: true,
              ratelimitApiDuration: true,
              allocatedCpuMillicoresTotal: true,
              allocatedMemoryMibTotal: true,
              allocatedStorageMibTotal: true,
              maxCpuMillicoresPerInstance: true,
              maxMemoryMibPerInstance: true,
              maxStorageMibPerInstance: true,
              maxConcurrentBuilds: true,
              maxReplicasPerRegion: true,
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
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      });
    } catch (_error) {
      console.debug("Workspace query failed in context creation");
      ws = undefined;
    }
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      // Recorded as `remote_ip` on every audit log, so it must be an address we trust rather than
      // whatever the client put in a forwarding header.
      location: getClientIp(req.headers) ?? "unknown",
    },
    user: authResult.userId
      ? {
          id: authResult.userId,
          // Profile from the sealed session cookie; saves provider API calls
          // for procedures that only need the signed-in user's profile.
          profile: authResult.user ?? null,
        }
      : null,
    // Billing state lives on the workspace_billing relation (the columns were
    // dropped from workspaces). Surface it under the legacy workspace field names
    // so existing ctx.workspace.<field> reads keep working against the billing
    // row. Writers target workspaceBilling directly.
    workspace: ws
      ? {
          ...ws,
          tier: ws.billing?.tier ?? "Free",
          stripeCustomerId: ws.billing?.stripeCustomerId ?? null,
          // Subscription ids now live one-per-product in billing_subscriptions;
          // flatten them back to the two legacy field names read across the app.
          ...subscriptionIdsByProduct(ws.billingSubscriptions ?? []),
          deployPlan: ws.billing?.plan ?? null,
          deployPlanOverride: ws.billing?.planOverride ?? null,
          deploySpendBudgetCents: ws.billing?.spendBudgetCents ?? null,
          deploySpendBudgetStop: ws.billing?.spendBudgetStop ?? false,
          deploySpendSuspended: ws.billing?.spendSuspended ?? false,
        }
      : ws,
    tenant: authResult.orgId
      ? {
          id: authResult.orgId,
          role: authResult.role,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
