import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { ratelimitLogEnrichment } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";

const RatelimitLogEnrichmentResponse = z.object({
  enrichment: z.array(ratelimitLogEnrichment),
});

export const queryRatelimitLogEnrichment = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      requestIds: z.array(z.string()).max(100),
      startTime: z.number().int(),
      endTime: z.number().int(),
    }),
  )
  .output(RatelimitLogEnrichmentResponse)
  .query(async ({ ctx, input }) => {
    if (input.requestIds.length === 0) {
      return { enrichment: [] };
    }

    const result = await clickhouse.ratelimits.logEnrichment({
      workspaceId: ctx.workspace.id,
      requestIds: input.requestIds,
      startTime: input.startTime,
      endTime: input.endTime,
    });

    if (result.err) {
      console.warn("Clickhouse ratelimit log enrichment query failed", {
        error: result.err.message,
        requestIds: input.requestIds,
      });
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch log enrichment data.",
      });
    }

    return { enrichment: result.val };
  });
