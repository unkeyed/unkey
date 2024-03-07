import { Ok, Result } from "@unkey/error";
import { RateLimiter, RatelimitError, RatelimitRequest, RatelimitResponse } from "./interface";

export class NoopRateLimiter implements RateLimiter {
  public async limit(req: RatelimitRequest): Promise<Result<RatelimitResponse, RatelimitError>> {
    console.log("noop limit", req);
    return Ok({ current: 0, pass: true, reset: 0 });
  }
}
