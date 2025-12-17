import { clickhouse } from "@/lib/clickhouse";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

const getLastUsedInput = z.object({
  namespaceId: z.string(),
  identifier: z.string(),
});

const getLastUsedOutput = z.object({
  identifier: z.string(),
  lastUsed: z.number().nullable(),
});

export const queryRatelimitLastUsed = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(getLastUsedInput)
  .output(getLastUsedOutput)
  .query(async ({ input, ctx }) => {
    try {
      const lastUsed = await clickhouse.ratelimits.latest({
        workspaceId: ctx.workspace.id,
        namespaceId: input.namespaceId,
        identifier: [input.identifier],
        limit: 1,
      });

      return {
        identifier: input.identifier,
        lastUsed: lastUsed.val?.at(0)?.time ?? null,
      };
    } catch (error) {
      console.error("Failed to fetch last used time", JSON.stringify(error));
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch last used time",
      });
    }
  });
