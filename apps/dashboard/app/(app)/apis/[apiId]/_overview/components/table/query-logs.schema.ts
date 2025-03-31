import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { MAX_KEYID_COUNT } from "../charts/bar-chart/query-timeseries.schema";

export const sortFields = z.enum(["time", "valid", "invalid"]);
export type SortFields = z.infer<typeof sortFields>;

export const keysQueryOverviewLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  apiId: z.string(),
  since: z.string(),
  cursor: z
    .object({
      requestId: z.string().nullable(),
      time: z.number().nullable(),
    })
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
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .max(MAX_KEYID_COUNT)
    .optional()
    .nullable(),
  names: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .optional()
    .nullable(),
  identities: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
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

export type KeysQueryOverviewLogsPayload = z.infer<typeof keysQueryOverviewLogsPayload>;
