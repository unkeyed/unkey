import { z } from "zod";

export const limitRequestSchema = z.object({
  /**
   * For legacy key-based credits stored in keys.remaining
   */
  keyId: z.string().optional(),
  /**
   * For new credits system stored in credits table.
   * When present, this takes precedence over keyId.
   */
  creditId: z.string().optional(),
  cost: z.number(),
});
export type LimitRequest = z.infer<typeof limitRequestSchema>;

export const limitResponseSchema = z.object({
  valid: z.boolean(),
  remaining: z.number().optional(),
});
export type LimitResponse = z.infer<typeof limitResponseSchema>;

export const revalidateRequestSchema = z.object({
  keyId: z.string().optional(),
  creditId: z.string().optional(),
});
export type RevalidateRequest = z.infer<typeof revalidateRequestSchema>;

export interface UsageLimiter {
  limit: (req: LimitRequest) => Promise<LimitResponse>;
  revalidate: (req: RevalidateRequest) => Promise<void>;
}
