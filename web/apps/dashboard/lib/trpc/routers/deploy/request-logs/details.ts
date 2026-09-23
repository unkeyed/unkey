import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  requestDetailsRequestSchema,
  requestLogsResponseSchema,
} from "@unkey/clickhouse/src/frontline";
import { z } from "zod";

export const queryRequestDetails = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(requestDetailsRequestSchema.omit({ workspaceId: true }))
  .output(z.object({ log: requestLogsResponseSchema.nullable() }))
  .query(async ({ ctx, input }) => {
    const result = await clickhouse.frontline.requestDetails({
      ...input,
      workspaceId: ctx.workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch request details from ClickHouse.",
      });
    }

    return { log: result.val[0] ?? null };
  });
