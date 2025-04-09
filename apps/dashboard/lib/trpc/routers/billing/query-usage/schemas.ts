import { z } from "zod";

export const queryUsageResponse = z.object({
  billableRatelimits: z.number(),
  billableVerifications: z.number(),
  billableTotal: z.number(),
});

export type UsageResponse = z.infer<typeof queryUsageResponse>;
