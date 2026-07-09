import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const keyDetailsLogsPayload = z.object({
  limit: z.int().min(1).max(100),
  startTime: z.int(),
  endTime: z.int(),
  keyspaceId: z.string(),
  keyId: z.string(),
  since: z.string(),
  // min(1) keeps the derived offset (page - 1) * limit non-negative in the query builder.
  page: z.int().min(1).optional().default(1),
  tags: z
    .array(
      z.object({
        value: z.string(),
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      }),
    )
    .optional()
    .nullable(),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .optional()
    .nullable(),
});

export type KeyDetailsLogsPayload = z.infer<typeof keyDetailsLogsPayload>;
