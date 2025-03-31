import { z } from "zod";
import { ratelimitFilterOperatorEnum } from "../../filters.schema";

export const ratelimitQueryLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  namespaceId: z.string(),
  since: z.string(),
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
  requestIds: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  status: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.enum(["blocked", "passed"]),
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

export type RatelimitQueryLogsPayload = z.infer<typeof ratelimitQueryLogsPayload>;
