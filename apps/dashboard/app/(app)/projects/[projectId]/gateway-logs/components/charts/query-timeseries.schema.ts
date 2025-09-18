import { z } from "zod";
import { gatewayLogsFilterOperatorEnum } from "../../gateway-logs-filters.schema";

export const queryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  path: z
    .object({
      filters: z.array(
        z.object({
          operator: gatewayLogsFilterOperatorEnum,
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
});
