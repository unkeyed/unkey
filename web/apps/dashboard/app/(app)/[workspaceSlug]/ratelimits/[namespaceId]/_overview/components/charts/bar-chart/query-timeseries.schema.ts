import { z } from "zod";
import { ratelimitOverviewFilterOperatorEnum } from "../../../filters.schema";

export const ratelimitOverviewQueryTimeseriesPayload = z.object({
  startTime: z.int(),
  endTime: z.int(),
  since: z.string(),
  namespaceId: z.string(),
  identifiers: z
    .object({
      filters: z.array(
        z.object({
          operator: ratelimitOverviewFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
});

export type RatelimitOverviewQueryTimeseriesPayload = z.infer<
  typeof ratelimitOverviewQueryTimeseriesPayload
>;
