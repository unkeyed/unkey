import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const identityQueryTimeseriesPayload = z.object({
  identityId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  granularity: z.enum([
    "minute",
    "fiveMinutes",
    "fifteenMinutes",
    "thirtyMinutes",
    "hour",
    "twoHours",
    "fourHours",
    "sixHours",
    "twelveHours",
    "day",
    "threeDays",
    "week",
    "month",
    "quarter",
  ]),
  since: z.string().optional(),
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

export type IdentityQueryTimeseriesPayload = z.infer<typeof identityQueryTimeseriesPayload>;
