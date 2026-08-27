import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

const lookbackMs = 24 * 60 * 60 * 1000;
const recentErrorsLimit = 20;

export const getRecentLogdrainErrors = workspaceProcedure
  .input(z.object({ drainId: z.string().min(1) }))
  .query(async ({ ctx, input }) => {
    const drain = await db.query.logdrains.findFirst({
      where: and(
        eq(schema.logdrains.id, input.drainId),
        eq(schema.logdrains.workspaceId, ctx.workspace.id),
      ),
      columns: { id: true },
    });
    if (!drain) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
    }

    const result = await clickhouse.logdrains.recentErrors({
      workspaceId: ctx.workspace.id,
      drainId: input.drainId,
      startMs: Date.now() - lookbackMs,
      limit: recentErrorsLimit,
    });
    if (!result.val) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch recent log drain errors",
      });
    }

    return result.val;
  });
