import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const identityLogsPayload = z.object({
  identityId: z.string(),
  limit: z.number().int().min(1).max(100).default(50),
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string().default(""),
  cursor: z.number().int().nullable().default(null),
  tags: z
    .array(
      z.object({
        value: z.string(),
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      }),
    )
    .nullable()
    .default(null),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .nullable()
    .default(null),
});

export type IdentityLogsPayload = z.infer<typeof identityLogsPayload>;
