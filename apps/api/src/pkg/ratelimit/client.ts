import { logger, metrics } from "@/pkg/global";
import { z } from "zod";
import { RateLimiter, RatelimitRequest, RatelimitResponse } from "./interface";

export class DurableRateLimiter implements RateLimiter {
  private readonly namespace: DurableObjectNamespace;
  private readonly domain: string;
  constructor(opts: {
    namespace: DurableObjectNamespace;

    domain?: string;
  }) {
    this.namespace = opts.namespace;
    this.domain = opts.domain ?? "unkey.dev";
  }

  public async limit(req: RatelimitRequest): Promise<RatelimitResponse> {
    const start = performance.now();
    const now = Date.now();
    const window = Math.floor(now / req.interval);
    const reset = (window + 1) * req.interval;

    const keyAndWindow = [req.keyId, window].join(":");

    try {
      const obj = this.namespace.get(this.namespace.idFromName(keyAndWindow));
      const url = `https://${this.domain}/limit`;
      const res = await obj.fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reset }),
      });

      const json = await res.json();
      const { current } = z.object({ current: z.number() }).parse(json);

      return {
        current,
        reset,
        pass: current <= req.limit,
      };
    } catch (e) {
      logger.error("ratelimit failed", { keyId: req.keyId, error: (e as Error).message });
      return {
        current: 0,
        reset,
        pass: false,
      };
    } finally {
      metrics.emit("metric.usagelimit", {
        latency: performance.now() - start,
        keyId: req.keyId,
      });
    }
  }
}
