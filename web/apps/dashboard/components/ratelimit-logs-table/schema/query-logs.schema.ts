import { ratelimitFilterOperatorEnum } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/filters.schema";
import { ratelimitLogsSort } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";

export const ratelimitQueryLogsPayload = z.object({
  limit: z.int().min(1).max(100),
  startTime: z.int(),
  endTime: z.int(),
  namespaceId: z.string(),
  since: z.string(),
  // min(1) keeps the derived offset (page - 1) * limit non-negative in the query builder.
  page: z.int().min(1).optional().default(1),
  sorts: z.array(ratelimitLogsSort).nullable().optional(),
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
});

export type RatelimitQueryLogsPayload = z.infer<typeof ratelimitQueryLogsPayload>;
