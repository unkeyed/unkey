import { sql } from "@unkey/db";
import { z } from "zod";
import { db } from "../db";
import * as workspaceFlags from "../flags";
import type { WorkspaceFlag } from "../flags";
import { subscriptionIdsByProduct } from "../stripe/billingSubscriptions";
import { resolveFlag } from "../workspace-flags";

const flagValues = z.record(
  z.string(),
  z.union([z.literal(0), z.literal(1)]).transform((value) => value === 1),
);

export async function loadWorkspace(
  orgId: string,
  requestedFlags: readonly WorkspaceFlag[] = Object.values(workspaceFlags),
) {
  const workspace = await db.query.workspaces.findFirst({
    where: (workspace, { and, eq, isNull }) =>
      and(eq(workspace.orgId, orgId), isNull(workspace.deletedAtM)),
    with: { limits: true, billing: true, billingSubscriptions: true },
    extras: {
      // Explicit identifiers keep Drizzle from dropping the outer table qualifier.
      flags: sql<unknown>`(
        SELECT COALESCE(JSON_OBJECTAGG(f.slug, COALESCE(o.value, f.default_value)), JSON_OBJECT())
        FROM flags AS f
        LEFT JOIN flag_overrides AS o
          ON o.flag_slug = f.slug
          AND o.workspace_id = ${sql.identifier("workspaces")}.${sql.identifier("id")}
        WHERE ${
          requestedFlags.length > 0
            ? sql`f.slug IN (${sql.join(
                requestedFlags.map((flag) => sql`${flag.slug}`),
                sql`, `,
              )})`
            : sql`FALSE`
        }
      )`.as("flags"),
    },
  });
  if (!workspace) {
    return undefined;
  }

  const { billing, billingSubscriptions } = workspace;
  const flags = flagValues.parse(
    typeof workspace.flags === "string" ? JSON.parse(workspace.flags) : workspace.flags,
  );
  return {
    ...workspace,
    flags: Object.fromEntries(
      requestedFlags.map((flag) => [flag.slug, resolveFlag({ workspace: { flags }, flag })]),
    ),
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
