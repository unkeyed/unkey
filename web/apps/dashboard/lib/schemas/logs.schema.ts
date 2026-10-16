import { z } from "zod";
import { logsFilterOperatorEnum } from "./logs.filter.schema";

// ### Timeseries

export type TimeseriesRequestSchema = z.infer<typeof timeseriesRequestSchema>;
export const timeseriesRequestSchema = z.object({
  startTime: z.int(),
  endTime: z.int(),
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
      filters: z
        .array(
          z.object({
            operator: z.literal("is"),
            value: z.string(),
          }),
        )
        .optional(),
      exclude: z.array(z.string()).optional(),
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
