import { and, count, db, eq, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { apis, keys } from "@unkey/db/src/schema";
import { z } from "zod";

const getKeyCountRequestSchema = z.object({
  apiId: z.string(),
});

const getKeyCountResponseSchema = z.object({
  count: z.number(),
});

export type GetKeyCountRequestSchema = z.TypeOf<typeof getKeyCountRequestSchema>;
export type GetKeyCountResponseSchema = z.TypeOf<typeof getKeyCountResponseSchema>;

export const getKeyCount = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(getKeyCountRequestSchema)
  .output(getKeyCountResponseSchema)
  .query(async ({ ctx, input }) => {
    const results = await db
      .select({
        count: count(keys.id),
      })
      .from(apis)
      .innerJoin(keys, eq(keys.keyAuthId, apis.keyAuthId))
      .where(
        and(
          eq(apis.id, input.apiId),
          eq(keys.workspaceId, ctx.workspace.id),
          isNull(keys.deletedAtM),
        ),
      )
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to retrieve the key count. Please try again or contact support@unkey.com",
        });
      });

    if (!results || results.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the specified API. Please try again or contact support@unkey.com.",
      });
    }

    return {
      count: results[0].count,
    };
  });
