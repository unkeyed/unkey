import type { Context } from "../hono/app";
import type { Metrics } from "../metrics";

type RatelimitRequest = {
  identifier: string;
  limit: number;
  duration: number;
  cost: number;
};

type RatelimitResponse = {
  limit: number;
  remaining: number;
  reset: number;
  success: boolean;
  current: number;
};

export class Agent {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly metrics: Metrics;

  constructor(baseUrl: string, token: string, metrics: Metrics) {
    this.baseUrl = baseUrl;
    this.token = token;
    this.metrics = metrics;
  }

  public async ratelimit(c: Context, req: RatelimitRequest): Promise<RatelimitResponse> {
    const start = performance.now();

    const res = await fetch(`${this.baseUrl}/ratelimit.v1.RatelimitService/Ratelimit`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.get("requestId"),
      },
      body: JSON.stringify(req),
    });
    const body = await res.json<Partial<RatelimitResponse>>();

    this.metrics.emit({
      metric: "metric.agent.latency",
      op: "ratelimit",
      latency: performance.now() - start,
    });
    return {
      limit: body.limit ?? 0,
      remaining: body.remaining ?? 0,
      reset: body.reset ?? 0,
      success: body.success ?? false,
      current: body.current ?? 0,
    };
  }
}
