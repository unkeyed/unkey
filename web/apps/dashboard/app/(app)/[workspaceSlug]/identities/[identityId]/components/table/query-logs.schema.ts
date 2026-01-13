import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const identityLogsPayload = z.object({
  identityId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  cursor: z.number().nullable().optional().nullable(),
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

export type IdentityLogsPayload = z.infer<typeof identityLogsPayload>;