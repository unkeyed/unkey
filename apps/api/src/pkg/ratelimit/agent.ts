import type { Logger } from "@unkey/worker-logging";
import type { Context } from "../hono/app";
import type { Metrics } from "../metrics";
import { instrumentedFetch } from "../util/instrument-fetch";

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
  private readonly logger: Logger;

  constructor(baseUrl: string, token: string, metrics: Metrics, logger: Logger) {
    this.baseUrl = baseUrl;
    this.token = token;
    this.metrics = metrics;
    this.logger = logger;
  }

  public async ratelimit(c: Context, req: RatelimitRequest): Promise<RatelimitResponse> {
    const start = performance.now();
    const url = `${this.baseUrl}/ratelimit.v1.RatelimitService/Ratelimit`;
    const res = await instrumentedFetch(c)(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.get("requestId"),
      },
      body: JSON.stringify({
        identifier: req.identifier,
        limit: req.limit,
        duration: req.duration,
        cost: req.cost,
      }),
    }).catch((err) => {
      console.error("Error in ratelimit", url, err);

      throw err;
    });

    const body = await res.text();

    if (!res.ok) {
      this.logger.error("Error in ratelimit", {
        url,
        status: res.status,
        body,
      });
    }

    const json = JSON.parse(body) as Partial<RatelimitResponse>;

    this.metrics.emit({
      metric: "metric.agent.latency",
      op: "ratelimit",
      latency: performance.now() - start,
    });
    return {
      limit: json.limit ?? 0,
      remaining: json.remaining ?? 0,
      reset: json.reset ?? 0,
      success: json.success ?? true,
      current: json.current ?? 0,
    };
  }
}
