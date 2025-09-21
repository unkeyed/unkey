import { verificationQueryTimeseriesPayload } from "@/app/(app)/[workspace]/apis/_components/hooks/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";

import { transformVerificationFilters } from "./utils";

export const queryVerificationTimeseries = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(verificationQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const { params: transformedInputs, granularity } = transformVerificationFilters(input);

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: input.keyspaceId,
    });

    return {
      timeseries: result,
      granularity,
    };
  });
