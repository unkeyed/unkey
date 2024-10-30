import { z } from "zod";

export const limitRequestSchema = z.object({
  keyId: z.string(),
});
export type LimitRequest = z.infer<typeof limitRequestSchema>;

export const limitResponseSchema = z.object({
  valid: z.boolean(),
  remaining: z.number().optional(),
});
export type LimitResponse = z.infer<typeof limitResponseSchema>;

export const revalidateRequestSchema = z.object({
  keyId: z.string(),
});
export type RevalidateRequest = z.infer<typeof revalidateRequestSchema>;

export interface UsageLimiter {
  limit: (req: LimitRequest) => Promise<LimitResponse>;
  revalidate: (req: RevalidateRequest) => Promise<void>;
}
