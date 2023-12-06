import { logger, metrics } from "@/pkg/global";
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
  constructor(opts: {
    namespace: DurableObjectNamespace;

    domain?: string;
  }) {
    this.namespace = opts.namespace;
    this.domain = opts.domain ?? "unkey.dev";
  }

  public async limit(req: LimitRequest): Promise<LimitResponse> {
    const start = performance.now();

    try {
      const obj = this.namespace.get(this.namespace.idFromName(req.keyId));
      const url = `https://${this.domain}/limit`;
      const res = await obj.fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      return limitResponseSchema.parse(await res.json());
    } catch (e) {
      logger.error("usagelimit failed", { error: e });
      return { valid: false };
    } finally {
      metrics.emit("metric.usagelimit", {
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
