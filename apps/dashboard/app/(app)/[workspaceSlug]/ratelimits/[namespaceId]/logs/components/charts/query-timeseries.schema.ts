import { z } from "zod";
import { ratelimitFilterOperatorEnum } from "../../filters.schema";

export const ratelimitQueryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  namespaceId: z.string(),
  identifiers: z
    .object({
      filters: z.array(
        z.object({
          operator: ratelimitFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
});

export type RatelimitQueryTimeseriesPayload = z.infer<typeof ratelimitQueryTimeseriesPayload>;
