import { Ok, type Result } from "@unkey/error";
import type { Context } from "hono";
import type { RateLimiter, RatelimitError, RatelimitRequest, RatelimitResponse } from "./interface";

export class NoopRateLimiter implements RateLimiter {
  public async limit(
    _c: Context,
    _req: RatelimitRequest,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    return Ok({ current: 0, passed: true, reset: 0, remaining: 0, triggered: null });
  }
  public async multiLimit(
    _c: Context,
    _req: Array<RatelimitRequest>,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    return Ok({ current: 0, passed: true, reset: 0, remaining: 0, triggered: null });
  }
}
