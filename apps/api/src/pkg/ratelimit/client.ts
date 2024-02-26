import { z } from "zod";
import { Logger } from "../logging";
import { Metrics } from "../metrics";
import { RateLimiter, RatelimitRequest, RatelimitResponse } from "./interface";

export class DurableRateLimiter implements RateLimiter {
  private readonly namespace: DurableObjectNamespace;
  private readonly domain: string;
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  constructor(opts: {
    namespace: DurableObjectNamespace;

    domain?: string;
    logger: Logger;
    metrics: Metrics;
  }) {
    this.namespace = opts.namespace;
    this.domain = opts.domain ?? "unkey.dev";
    this.logger = opts.logger;
    this.metrics = opts.metrics;
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
      const res = await obj
        .fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reset }),
        })
        .catch(async (e) => {
          this.logger.warn("calling the ratelimit DO failed, retrying ...", {
            keyId: req.keyId,
            error: (e as Error).message,
          });
          return await obj.fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ reset }),
          });
        });

      const json = await res.json();
      const { current } = z.object({ current: z.number() }).parse(json);

      return {
        current,
        reset,
        pass: current <= req.limit,
      };
    } catch (e) {
      this.logger.error("ratelimit failed", { keyId: req.keyId, error: (e as Error).message });
      return {
        current: 0,
        reset,
        pass: false,
      };
    } finally {
      this.metrics.emit({
        metric: "metric.usagelimit",
        latency: performance.now() - start,
        keyId: req.keyId,
      });
    }
  }
}
