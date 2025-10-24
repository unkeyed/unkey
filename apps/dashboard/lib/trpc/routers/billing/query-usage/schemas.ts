import { z } from "zod";

export const queryUsageResponse = z.object({
  billableRatelimits: z.number().default(0),
  billableVerifications: z.number().default(0),
  billableTotal: z.number().default(0),
});

export type UsageResponse = z.infer<typeof queryUsageResponse>;
