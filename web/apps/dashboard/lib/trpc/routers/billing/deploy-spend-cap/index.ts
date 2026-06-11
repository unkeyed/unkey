import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { formatDollars } from "@/lib/fmt";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspaceAdmin, withRatelimit, workspaceProcedure } from "../../../trpc";

/**
 * Upper bound on a cap: $10M/month. Far above any real bill; exists only so
 * a typo cannot store a nonsense value.
 */
const MAX_CAP_CENTS = 1_000_000_000;

/**
 * A single cap value: whole dollars only (the dashboard form takes dollars
 * and converts, cent precision on a cap would be noise), or null = not set.
 */
const capCents = z.number().int().positive().max(MAX_CAP_CENTS).multipleOf(100).nullable();

/**
 * The workspace's monthly Compute spend caps. Soft = notify when crossed,
 * hard = stop workloads. NULL = not set, independently per cap. v1 stores the
 * preferences only: nothing enforces them, sends notifications, or clamps
 * workloads yet.
 */
export const getDeploySpendCaps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(({ ctx }) => ({
    softCapCents: ctx.workspace.deploySpendCapSoftCents ?? null,
    hardCapCents: ctx.workspace.deploySpendCapHardCents ?? null,
  }));

/** Sets (or clears, with null) the monthly Compute spend caps. */
export const setDeploySpendCaps = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(z.object({ softCapCents: capCents, hardCapCents: capCents }))
  .mutation(async ({ ctx, input }) => {
    // A soft cap above the hard cap would notify about a level the hard cap
    // never lets the workspace reach.
    if (
      input.softCapCents !== null &&
      input.hardCapCents !== null &&
      input.softCapCents > input.hardCapCents
    ) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "The soft cap must not exceed the hard cap.",
      });
    }

    await db
      .update(schema.workspaces)
      .set({
        deploySpendCapSoftCents: input.softCapCents,
        deploySpendCapHardCents: input.hardCapCents,
      })
      .where(eq(schema.workspaces.id, ctx.workspace.id));

    const describe = (cents: number | null) =>
      cents === null ? "none" : `${formatDollars(cents)}/month`;
    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description: `Set the Compute spend caps: soft ${describe(input.softCapCents)}, hard ${describe(input.hardCapCents)}.`,
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });

    await invalidateWorkspaceCache(ctx.tenant.id);

    return { softCapCents: input.softCapCents, hardCapCents: input.hardCapCents };
  });
