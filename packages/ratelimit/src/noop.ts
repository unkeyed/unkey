import type { Ratelimiter } from "./interface";
import type { LimitOptions, RatelimitResponse } from "./types";

export class NoopRatelimit implements Ratelimiter {
  public limit(_identifier: string, _opts?: LimitOptions): Promise<RatelimitResponse> {
    return Promise.resolve({
      limit: 0,
      remaining: 0,
      reset: 0,
      success: true,
    });
  }
}
