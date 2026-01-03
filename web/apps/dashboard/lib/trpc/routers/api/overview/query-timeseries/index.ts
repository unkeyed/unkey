import { verificationQueryTimeseriesPayload } from "@/app/(app)/[workspaceSlug]/apis/_components/hooks/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

import { transformVerificationFilters } from "./utils";

export const queryVerificationTimeseries = workspaceProcedure
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
