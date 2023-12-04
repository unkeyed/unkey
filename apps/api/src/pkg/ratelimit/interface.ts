import { z } from "zod";

export const ratelimitRequestSchema = z.object({
  keyId: z.string(),
  limit: z.number().int(),
  interval: z.number().int(),
});
export type RatelimitRequest = z.infer<typeof ratelimitRequestSchema>;

export const ratelimitResponseSchema = z.object({
  current: z.number(),
  reset: z.number(),
  pass: z.boolean(),
});
export type RatelimitResponse = z.infer<typeof ratelimitResponseSchema>;

export interface RateLimiter {
  limit: (req: RatelimitRequest) => Promise<RatelimitResponse>;
}
