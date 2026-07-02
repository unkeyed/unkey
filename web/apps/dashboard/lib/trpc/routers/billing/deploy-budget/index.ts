import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { formatDollars } from "@/lib/fmt";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspaceAdmin, withRatelimit, workspaceProcedure } from "../../../trpc";

/**
 * Upper bound on the budget: $10M/month. Far above any real bill; exists only
 * so a typo cannot store a nonsense value.
 */
const MAX_BUDGET_CENTS = 1_000_000_000;

/**
 * The workspace's monthly Compute spend budget. NULL = no budget. Email
 * alerts fire at fixed percentages of the budget (50/75/100); stopAtBudget
 * additionally stops workloads when month-to-date usage spend reaches it.
 * v1 stores the preferences only: nothing alerts or stops yet.
 */
export const getDeployBudget = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(({ ctx }) => ({
    budgetCents: ctx.workspace.deploySpendBudgetCents ?? null,
    stopAtBudget: ctx.workspace.deploySpendBudgetStop,
    // True while the spend cap has paused this workspace's compute; the
    // dashboard surfaces it so an admin understands why compute is offline.
    suspended: ctx.workspace.deploySpendSuspended,
    // The period's included usage credit, in cents. The spend cap measures
    // net-of-credit overage (max(0, gross usage - this)), so the dashboard
    // subtracts it too and the meter matches what the backend enforces. NULL
    // means not yet known (no invoice event has persisted it); the checker
    // skips alerts and enforcement then, so the dashboard must render that
    // state rather than pretend the credit is zero and show gross as net.
    includedCreditCents: ctx.workspace.deployIncludedCreditCents ?? null,
  }));

/**
 * Sets (or clears, with null) the monthly Compute spend budget. Whole dollars
 * only: the dashboard form takes dollars and converts, so cent precision on a
 * budget would be noise.
 */
export const setDeployBudget = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      budgetCents: z.number().int().positive().max(MAX_BUDGET_CENTS).multipleOf(100).nullable(),
      stopAtBudget: z.boolean(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // A stop without a budget has no trigger point; reject instead of storing
    // a toggle that silently does nothing.
    if (input.stopAtBudget && input.budgetCents === null) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Set a budget to stop workloads at.",
      });
    }

    await db
      .update(schema.workspaces)
      .set({
        deploySpendBudgetCents: input.budgetCents,
        deploySpendBudgetStop: input.stopAtBudget,
      })
      .where(eq(schema.workspaces.id, ctx.workspace.id));

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description:
        input.budgetCents === null
          ? "Removed the Compute spend budget."
          : `Set the Compute spend budget to ${formatDollars(input.budgetCents)}/month (stop workloads: ${input.stopAtBudget ? "on" : "off"}).`,
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });

    return { budgetCents: input.budgetCents, stopAtBudget: input.stopAtBudget };
  });
