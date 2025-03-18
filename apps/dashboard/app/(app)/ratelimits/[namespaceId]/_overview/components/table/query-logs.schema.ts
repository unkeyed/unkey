import { z } from "zod";
import { ratelimitOverviewFilterOperatorEnum } from "../../filters.schema";

export const sortFields = z.enum(["time", "avg_latency", "p99_latency", "blocked", "passed"]);
export const ratelimitQueryOverviewLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  namespaceId: z.string(),
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
  sorts: z
    .array(
      z.object({
        column: sortFields,
        direction: z.enum(["asc", "desc"]),
      }),
    )
    .optional()
    .nullable(),
});

export type RatelimitQueryOverviewLogsPayload = z.infer<typeof ratelimitQueryOverviewLogsPayload>;
