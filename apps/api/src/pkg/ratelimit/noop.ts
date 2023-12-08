import { RateLimiter, RatelimitRequest, RatelimitResponse } from "./interface";

export class NoopRateLimiter implements RateLimiter {
  public async limit(req: RatelimitRequest): Promise<RatelimitResponse> {
    console.log("noop limit", req);
    return { current: 0, pass: true, reset: 0 };
  }
}
