import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { keysOverviewFilterOperatorEnum } from "../../../filters.schema";

export const MAX_KEYID_COUNT = 15;
export const keysOverviewQueryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  apiId: z.string(),
  keyIds: z
    .object({
      filters: z
        .array(
          z.object({
            operator: keysOverviewFilterOperatorEnum,
            value: z.string(),
          }),
        )
        .max(MAX_KEYID_COUNT),
    })
    .nullable(),
  names: z
    .object({
      filters: z.array(
        z.object({
          operator: keysOverviewFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  identities: z
    .object({
      filters: z.array(
        z.object({
          operator: keysOverviewFilterOperatorEnum,
          value: z.string(),
        }),
      ),
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

export type KeysOverviewQueryTimeseriesPayload = z.infer<typeof keysOverviewQueryTimeseriesPayload>;
