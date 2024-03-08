import { Err, Ok, Result } from "@unkey/error";
import { z } from "zod";
import { Logger } from "../logging";
import { Metrics } from "../metrics";
import { RateLimiter, RatelimitError, RatelimitRequest, RatelimitResponse } from "./interface";

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

  public async limit(req: RatelimitRequest): Promise<Result<RatelimitResponse, RatelimitError>> {
    const start = performance.now();
    const now = Date.now();
    const window = Math.floor(now / req.interval);
    const reset = (window + 1) * req.interval;

    const keyAndWindow = [req.identifier, window].join(":");

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
            identifier: req.identifier,
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

      return Ok({
        current,
        reset,
        pass: current <= req.limit,
      });
    } catch (e) {
      const err = e as Error;
      this.logger.error("ratelimit failed", { identifier: req.identifier, error: err.message });
      return Err(new RatelimitError(err.message));
    } finally {
      this.metrics.emit({
        metric: "metric.ratelimit",
        latency: performance.now() - start,
        identifier: req.identifier,
        tier: "durable",
      });
    }
  }
}
