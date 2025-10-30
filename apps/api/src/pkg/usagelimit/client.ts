import type { Logger } from "@unkey/worker-logging";
import type { Metrics } from "../metrics";
import {
  type LimitRequest,
  type LimitResponse,
  type RevalidateRequest,
  type UsageLimiter,
  limitResponseSchema,
} from "./interface";

export class DurableUsageLimiter implements UsageLimiter {
  private readonly namespace: DurableObjectNamespace;
  private readonly domain: string;
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  private readonly requestId: string;
  constructor(opts: {
    namespace: DurableObjectNamespace;
    requestId: string;

    domain?: string;
    logger: Logger;
    metrics: Metrics;
  }) {
    this.requestId = opts.requestId;
    this.namespace = opts.namespace;
    this.domain = opts.domain ?? "unkey.dev";
    this.logger = opts.logger;
    this.metrics = opts.metrics;
  }

  private getStub(name: string): DurableObjectStub {
    return this.namespace.get(this.namespace.idFromName(name));
  }

  public async limit(req: LimitRequest): Promise<LimitResponse> {
    const start = performance.now();

    // Use creditId if present (new system), otherwise fall back to keyId (legacy)
    const identifier = req.creditId ?? req.keyId;
    if (!identifier) {
      this.logger.error("usagelimit called without keyId or creditId");
      return { valid: false };
    }

    try {
      const url = `https://${this.domain}/limit`;
      const res = await this.getStub(identifier)
        .fetch(url, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Unkey-Request-Id": this.requestId,
          },
          body: JSON.stringify(req),
        })
        .catch(async (e) => {
          this.logger.warn("calling the usagelimit DO failed, retrying ...", {
            keyId: req.keyId,
            creditId: req.creditId,
            error: (e as Error).message,
          });
          return await this.getStub(identifier).fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(req),
          });
        });
      return limitResponseSchema.parse(await res.json());
    } catch (e) {
      this.logger.error("usagelimit failed", {
        keyId: req.keyId,
        creditId: req.creditId,
        error: (e as Error).message,
      });
      return { valid: false };
    } finally {
      this.metrics.emit({
        metric: "metric.usagelimit",
        latency: performance.now() - start,
        keyId: req.keyId,
        creditId: req.creditId,
      });
    }
  }

  public async revalidate(req: RevalidateRequest): Promise<void> {
    const identifier = req.creditId ?? req.keyId;
    if (!identifier) {
      this.logger.error("revalidate called without keyId or creditId");
      return;
    }

    const obj = this.namespace.get(this.namespace.idFromName(identifier));
    const url = `https://${this.domain}/revalidate`;
    await obj.fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
  }
}
