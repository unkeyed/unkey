import { logsFilterOperatorEnum } from "@/app/(app)/logs/filters.schema";
import { log } from "@unkey/clickhouse/src/logs";
import { z } from "zod";

export type LogsRequestSchema = z.infer<typeof logsRequestSchema>;
export const logsRequestSchema = z.object({
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
  cursor: z.number().nullable().optional().nullable(),
});

export const logsResponseSchema = z.object({
  logs: z.array(log),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().optional(),
});

export type LogsResponseSchema = z.infer<typeof logsResponseSchema>;

// ### Timeseries

export type TimeseriesRequestSchema = z.infer<typeof timeseriesRequestSchema>;
export const timeseriesRequestSchema = z.object({
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
