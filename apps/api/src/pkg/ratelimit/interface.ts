import { BaseError, Result } from "@unkey/error";
import { Context } from "hono";
import { z } from "zod";

export class RatelimitError extends BaseError {
  public readonly name = "RatelimitError";
  public readonly retry = false;
}

export const ratelimitRequestSchema = z.object({
  workspaceId: z.string(),
  namespaceId: z.string().optional(),
  identifier: z.string(),
  limit: z.number().int(),
  interval: z.number().int(),
  cost: z.number().int().min(1).default(1).optional(),
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
  reset: z.number(),
  pass: z.boolean(),
});
export type RatelimitResponse = z.infer<typeof ratelimitResponseSchema>;

export interface RateLimiter {
  limit: (c: Context, req: RatelimitRequest) => Promise<Result<RatelimitResponse, RatelimitError>>;
}
