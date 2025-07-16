import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { TAG_OPERATORS } from "../../../filters.schema";

export const MAX_KEYID_COUNT = 1;
export const keyDetailsQueryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  keyspaceId: z.string(),
  keyId: z.string(),
  tags: z
    .object({
      operator: z.enum(TAG_OPERATORS),
      value: z.string(),
    })
    .nullable(),
  outcomes: z
    .object({
      filters: z.array(
        z.object({
          value: z.enum(KEY_VERIFICATION_OUTCOMES),
          operator: z.literal("is"),
        }),
      ),
    })
    .nullable(),
});

export type KeyDetailsQueryTimeseriesPayload = z.infer<typeof keyDetailsQueryTimeseriesPayload>;
