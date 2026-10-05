import { and, eq, isNull, schema, sql } from "@unkey/db";
import { db } from "../db";
import { subscriptionIdsByProduct } from "../stripe/billingSubscriptions";

export async function loadWorkspace(orgId: string) {
  const rows = await db
    .select({
      workspace: schema.workspaces,
      limits: schema.limits,
      billing: schema.workspaceBilling,
      subscription: schema.billingSubscriptions,
      flag: { slug: schema.flags.slug, defaultValue: schema.flags.defaultValue },
      override: { value: schema.workspaceFlagOverrides.value },
    })
    .from(schema.workspaces)
    .leftJoin(schema.limits, eq(schema.limits.workspaceId, schema.workspaces.id))
    .leftJoin(
      schema.workspaceBilling,
      eq(schema.workspaceBilling.workspaceId, schema.workspaces.id),
    )
    .leftJoin(
      schema.billingSubscriptions,
      eq(schema.billingSubscriptions.workspaceId, schema.workspaces.id),
    )
    .leftJoin(schema.flags, sql`true`)
    .leftJoin(
      schema.workspaceFlagOverrides,
      and(
        eq(schema.workspaceFlagOverrides.workspaceId, schema.workspaces.id),
        eq(schema.workspaceFlagOverrides.flagId, schema.flags.id),
      ),
    )
    .where(and(eq(schema.workspaces.orgId, orgId), isNull(schema.workspaces.deletedAtM)));

  const first = rows[0];
  if (!first) {
    return undefined;
  }

  const flags = new Map<string, boolean>();
  const subscriptions = new Map<string, typeof schema.billingSubscriptions.$inferSelect>();
  for (const row of rows) {
    if (row.flag) {
      flags.set(row.flag.slug, row.override?.value ?? row.flag.defaultValue);
    }
    if (row.subscription) {
      subscriptions.set(row.subscription.product, row.subscription);
    }
  }
  const billingSubscriptions = [...subscriptions.values()];
  const { workspace, limits, billing } = first;
  return {
    ...workspace,
    limits,
    billing,
    billingSubscriptions,
    flags: Object.fromEntries(flags),
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
