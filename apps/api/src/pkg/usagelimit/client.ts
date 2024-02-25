import { Logger } from "../logging";
import { Metrics } from "../metrics";
import {
  LimitRequest,
  LimitResponse,
  RevalidateRequest,
  UsageLimiter,
  limitResponseSchema,
} from "./interface";

export class DurableUsageLimiter implements UsageLimiter {
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

  public async limit(req: LimitRequest): Promise<LimitResponse> {
    const start = performance.now();

    try {
      const obj = this.namespace.get(this.namespace.idFromName(req.keyId));
      const url = `https://${this.domain}/limit`;
      const res = await obj
        .fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        })
        .catch(async (e) => {
          this.logger.warn("calling the usagelimit DO failed, retrying ...", {
            keyId: req.keyId,
            error: (e as Error).message,
          });
          return await obj.fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(req),
          });
        });
      return limitResponseSchema.parse(await res.json());
    } catch (e) {
      this.logger.error("usagelimit failed", { keyId: req.keyId, error: (e as Error).message });
      return { valid: false };
    } finally {
      this.metrics.emit({
        metric: "metric.usagelimit",
        latency: performance.now() - start,
        keyId: req.keyId,
      });
    }
  }

  public async revalidate(req: RevalidateRequest): Promise<void> {
    const obj = this.namespace.get(this.namespace.idFromName(req.keyId));
    const url = `https://${this.domain}/revalidate`;
    await obj.fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
  }
}
