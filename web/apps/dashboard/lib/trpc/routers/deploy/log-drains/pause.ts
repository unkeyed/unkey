import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

// Pause flips `enabled = false` on the config row. The coordinator filters
// on enabled in its tick query, so pause takes effect on the next tick.
// We do NOT touch paused_reason — that field tracks auto-pause from the
// coordinator's failure threshold. Manual pause is a separate operator
// concern and should be distinguishable from "the provider is down".
export const pauseLogDrain = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .mutation(async ({ ctx, input }) => {
    try {
      await assertOwned(ctx.workspace.id as string, input.id);
      await db
        .update(schema.logDrains)
        .set({ enabled: false, updatedAt: Date.now() })
        .where(eq(schema.logDrains.id, input.id));
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to pause log drain",
      });
    }
  });

// Resume re-enables the drain AND clears any auto-pause state. Customers
// hit Resume after fixing whatever the provider was complaining about
// (token rotation, dataset rename, quota), so wiping last_error and
// consecutive_failures gives them a clean slate without forcing a CLI
// query.
export const resumeLogDrain = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const now = Date.now();
    try {
      await assertOwned(ctx.workspace.id as string, input.id);
      await db.transaction(async (tx) => {
        await tx
          .update(schema.logDrains)
          .set({ enabled: true, updatedAt: now })
          .where(eq(schema.logDrains.id, input.id));

        // Clear any prior auto-pause so the next tick treats this drain
        // as fresh. Done via raw SQL so we can express "set lastError =
        // NULL" cleanly; drizzle's typed update wants explicit nulls
        // which drop into the binlog as updates we do not need.
        await tx.execute(sql`
          UPDATE log_drain_state
          SET last_error = NULL,
              consecutive_failures = 0,
              paused_reason = NULL,
              updated_at = ${now}
          WHERE drain_id = ${input.id}
        `);
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to resume log drain",
      });
    }
  });

// assertOwned is the cross-tenant guard. Without it, an attacker who
// guessed a drain ID could pause any drain workspace-wide. We also check
// soft-delete so a re-enable on a deleted drain returns NOT_FOUND.
async function assertOwned(workspaceId: string, drainId: string): Promise<void> {
  const row = await db.query.logDrains.findFirst({
    where: and(
      eq(schema.logDrains.id, drainId),
      eq(schema.logDrains.workspaceId, workspaceId),
      isNull(schema.logDrains.deletedAt),
    ),
    columns: { id: true },
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
  }
}
