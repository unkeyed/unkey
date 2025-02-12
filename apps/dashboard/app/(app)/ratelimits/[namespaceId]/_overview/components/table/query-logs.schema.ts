import { z } from "zod";
import { ratelimitOverviewFilterOperatorEnum } from "../../filters.schema";

export const ratelimitQueryOverviewLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  namespaceId: z.string(),
  since: z.string(),
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
  cursor: z
    .object({
      requestId: z.string().nullable(),
      time: z.number().nullable(),
    })
    .optional()
    .nullable(),
});

export type RatelimitQueryOverviewLogsPayload = z.infer<typeof ratelimitQueryOverviewLogsPayload>;
