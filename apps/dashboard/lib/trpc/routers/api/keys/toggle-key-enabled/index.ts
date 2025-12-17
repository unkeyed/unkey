import { and, db, eq, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { keys } from "@unkey/db/src/schema";
import { z } from "zod";

export const enableKey = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      keyId: z.string().min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const keyToEnable = await db.query.keys
      .findFirst({
        where: and(
          eq(keys.id, input.keyId),
          eq(keys.workspaceId, ctx.workspace.id),
          isNull(keys.deletedAtM),
        ),
        columns: {
          id: true,
          enabled: true,
        },
      })
      .catch((err) => {
        console.error("Database error finding key for enabling:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to check key status. Please try again.",
        });
      });

    if (!keyToEnable) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Key not found or you don't have access to modify it.",
      });
    }

    if (keyToEnable.enabled) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Key is already enabled.",
      });
    }

    try {
      await db
        .update(keys)
        .set({ enabled: true })
        .where(and(eq(keys.id, input.keyId), eq(keys.workspaceId, ctx.workspace.id)));
    } catch (err) {
      console.error("Database error enabling key:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to enable the key. Please try again.",
      });
    }

    return { success: true };
  });
