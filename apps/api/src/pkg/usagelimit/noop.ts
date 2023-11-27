import { LimitRequest, LimitResponse, RevalidateRequest, UsageLimiter } from "./interface";

export class NoopUsageLimiter implements UsageLimiter {
  public async limit(_req: LimitRequest): Promise<LimitResponse> {
    return { valid: true, remaining: -1 };
  }

  public async revalidate(_req: RevalidateRequest): Promise<void> {}
}
