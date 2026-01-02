import { BaseError, type Result } from "@unkey/error";
import type { Context } from "hono";
import { z } from "zod";

export class RatelimitError extends BaseError {
  public readonly retry = false;
  public readonly name = RatelimitError.name;
}

export const ratelimitRequestSchema = z.object({
  name: z.string(),
  workspaceId: z.string(),
  namespaceId: z.string().optional(),
  identifier: z.string(),
  limit: z.number().int(),
  interval: z.number().int(),
  /**
   * Setting cost to 0 should not change anything but return the current limit
   */
  cost: z.number().int().min(0).default(1).optional(),
  /**
   * Add an arbitrary string to the durable object name.
   * We use this to do limiting at the edge for root keys by adding the cloudflare colo
   */
  shard: z.string().optional(),
  async: z.boolean().optional(),
});
export type RatelimitRequest = z.infer<typeof ratelimitRequestSchema>;

export const ratelimitResponseSchema = z.object({
  current: z.number(),
  remaining: z.number(),
  reset: z.number(),
  passed: z.boolean(),
  /**
   * The name of the limit that triggered a rejection
   */
  triggered: z.string().nullable(),
});
export type RatelimitResponse = z.infer<typeof ratelimitResponseSchema>;

export interface RateLimiter {
  limit: (c: Context, req: RatelimitRequest) => Promise<Result<RatelimitResponse, RatelimitError>>;
  multiLimit: (
    c: Context,
    req: Array<RatelimitRequest>,
  ) => Promise<Result<RatelimitResponse, RatelimitError>>;
}
