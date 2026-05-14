import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getApi } from "./api-query";

/**
 * Answers "has this API's keyspace ever recorded a key verification" — an
 * all-time, unbounded check, unlike the timeframe-scoped overview queries. The
 * Requests page uses it to decide whether to collapse to a single empty state
 * for brand-new APIs.
 */
export const keyspaceHasVerifications = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ apiId: z.string() }))
  .output(z.object({ hasData: z.boolean() }))
  .query(async ({ ctx, input }) => {
    const api = await getApi(input.apiId, ctx.workspace.id);
    if (!api || !api.keyAuth?.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or does not have key authentication enabled",
      });
    }

    const result = await clickhouse.api.keys.hasVerifications({
      workspaceId: ctx.workspace.id,
      keyspaceId: api.keyAuth.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when checking verification data from ClickHouse.",
      });
    }

    return { hasData: result.val.hasData };
  });
