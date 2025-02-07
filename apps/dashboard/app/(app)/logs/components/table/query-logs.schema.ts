import { z } from "zod";
import { logsFilterOperatorEnum } from "../../filters.schema";

export const queryLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  path: z
    .object({
      filters: z.array(
        z.object({
          operator: logsFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  host: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  method: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  requestId: z
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
          value: z.number(),
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
