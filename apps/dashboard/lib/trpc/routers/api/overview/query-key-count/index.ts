import { and, count, db, eq, isNull } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
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

export const getKeyCount = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(getKeyCountRequestSchema)
  .output(getKeyCountResponseSchema)
  .query(async ({ ctx, input }) => {
    const [result] = await db
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
      );

    return {
      count: result.count,
    };
  });
