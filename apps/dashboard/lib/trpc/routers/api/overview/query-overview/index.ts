import { fetchApiOverview } from "@/app/(app)/[workspaceSlug]/apis/actions";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";

import { TRPCError } from "@trpc/server";
import { apisOverviewResponse, queryApisOverviewPayload } from "./schemas";

export const queryApisOverview = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(queryApisOverviewPayload)
  .output(apisOverviewResponse)
  .query(async ({ ctx, input }) => {
    try {
      const result = await fetchApiOverview({
        workspaceId: ctx.workspace.id,
        limit: input.limit,
        cursor: input.cursor,
      });

      return result;
    } catch (error) {
      console.error("Something went wrong when fetching api overview list", JSON.stringify(error));
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch API overview",
      });
    }
  });
