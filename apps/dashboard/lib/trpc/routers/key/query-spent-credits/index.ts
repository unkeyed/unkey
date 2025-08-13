import { clickhouse } from "@/lib/clickhouse";
import { db, isNull } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const querySpentCreditsSchema = z.object({
  keyId: z.string(),
  keyspaceId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  outcomes: z
    .array(
      z.object({
        value: z.enum([
          "VALID",
          "RATE_LIMITED",
          "INSUFFICIENT_PERMISSIONS",
          "FORBIDDEN",
          "DISABLED",
          "EXPIRED",
          "USAGE_EXCEEDED",
        ]),
        operator: z.literal("is"),
      }),
    )
    .nullable()
    .optional(),
  tags: z
    .object({
      operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      value: z.string(),
    })
    .nullable()
    .optional(),
});

export const queryKeySpentCredits = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(querySpentCreditsSchema)
  .query(async ({ ctx, input }) => {
    // Verify the key belongs to the workspace
    const key = await db.query.keys
      .findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.keyAuthId, input.keyspaceId),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        columns: {
          id: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Key not found or does not belong to your workspace",
      });
    }

    const result = await clickhouse.verifications.spentCreditsTotal({
      workspaceId: ctx.workspace.id,
      keyspaceId: input.keyspaceId,
      keyId: input.keyId,
      startTime: input.startTime,
      endTime: input.endTime,
      outcomes: input.outcomes || null,
      tags: input.tags || null,
    });

    return {
      spentCredits: result.val?.[0]?.spent_credits ?? 0,
    };
  });
