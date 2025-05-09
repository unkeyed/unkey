import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const keyDetailsLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  keyspaceId: z.string(),
  keyId: z.string(),
  since: z.string(),
  cursor: z.number().nullable().optional().nullable(),
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
