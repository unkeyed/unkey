import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const identityLogsPayload = z.object({
  identityId: z.string(),
  limit: z.int().min(1).max(100).prefault(50),
  startTime: z.int(),
  endTime: z.int(),
  since: z.string().prefault(""),
  cursor: z.int().nullable().prefault(null),
  tags: z
    .array(
      z.object({
        value: z.string(),
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      }),
    )
    .nullable()
    .prefault(null),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .nullable()
    .prefault(null),
});

export type IdentityLogsPayload = z.infer<typeof identityLogsPayload>;
