import { and, db, eq, isNull, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

// Soft-delete (deleted_at = now). The coordinator's enabled-drains query
// already filters on `deleted_at IS NULL`, so a deleted row stops being
// fetched on the next tick without a foreground RPC. Credentials and
// state rows are left dangling intentionally — the FKs are weak by
// design (no ON DELETE), and keeping them around is useful if a customer
// ever needs to audit "what happened on drain X right before it was
// deleted".
export const deleteLogDrain = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const now = Date.now();
    try {
      const result = await db
        .update(schema.logDrains)
        .set({ deletedAt: now, enabled: false })
        .where(
          and(
            eq(schema.logDrains.id, input.id),
            eq(schema.logDrains.workspaceId, ctx.workspace.id as string),
            isNull(schema.logDrains.deletedAt),
          ),
        );
      // drizzle-mysql returns ResultSetHeader as the first element; use a
      // shape-tolerant access since the typing varies across drivers.
      const affected = (result as unknown as [{ affectedRows: number }])[0]?.affectedRows ?? 0;
      if (affected === 0) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
      }
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete log drain",
      });
    }
  });
