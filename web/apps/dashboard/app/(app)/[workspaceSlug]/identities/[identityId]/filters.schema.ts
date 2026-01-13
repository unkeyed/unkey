import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";

export const identityDetailsFilterFieldConfig = {
  tags: {
    operators: ["is", "contains", "startsWith", "endsWith"] as const,
  },
  outcomes: {
    operators: ["is"] as const,
    values: KEY_VERIFICATION_OUTCOMES,
  },
  startTime: {
    operators: ["is"] as const,
  },
  endTime: {
    operators: ["is"] as const,
  },
  since: {
    operators: ["is"] as const,
  },
} as const;

export const identityDetailsFilterValue = z.discriminatedUnion("field", [
  z.object({
    id: z.string(),
    field: z.literal("tags"),
    operator: z.enum(identityDetailsFilterFieldConfig.tags.operators),
    value: z.string(),
  }),
  z.object({
    id: z.string(),
    field: z.literal("outcomes"),
    operator: z.enum(identityDetailsFilterFieldConfig.outcomes.operators),
    value: z.enum(KEY_VERIFICATION_OUTCOMES),
  }),
  z.object({
    id: z.string(),
    field: z.literal("startTime"),
    operator: z.enum(identityDetailsFilterFieldConfig.startTime.operators),
    value: z.number(),
  }),
  z.object({
    id: z.string(),
    field: z.literal("endTime"),
    operator: z.enum(identityDetailsFilterFieldConfig.endTime.operators),
    value: z.number(),
  }),
  z.object({
    id: z.string(),
    field: z.literal("since"),
    operator: z.enum(identityDetailsFilterFieldConfig.since.operators),
    value: z.string(),
  }),
]);

export type IdentityDetailsFilterValue = z.infer<typeof identityDetailsFilterValue>;