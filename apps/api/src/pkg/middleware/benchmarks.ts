import type { Context, MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";
import type { Metrics } from "../metrics";

export function benchmarks(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    try {
      c.executionCtx.waitUntil(ping(c, "aws", "https://api-56140r9a.fctl.app/v1/liveness"));
      c.executionCtx.waitUntil(
        ping(c, "koyeb-all", "https://rich-mela-unkey-95820a9c.koyeb.app/v1/liveness"),
      );
      c.executionCtx.waitUntil(
        ping(c, "koyeb-us-east", "https://grubby-christabella-unkey-4e61f00b.koyeb.app/"),
      );
    } catch (e) {
      c.get("services").logger.warn("benchmark error", {
        error: (e as Error).message,
      });
    }
    return next();
  };
}
// "https://api-56140r9a.fctl.app/v1/liveness"

async function ping(c: Context, platform: string, url: string): Promise<void> {
  const start = performance.now();
  const res = await fetch(url);
  await res.text();

  c.get("services").metrics.emit({
    metric: "metric.server.latency",
    platform,
    // @ts-expect-error
    country: c.req.raw?.cf?.country ?? "unknown",
    // @ts-ignore
    continent: c.req.raw?.cf?.continent ?? "unknown",
    latency: performance.now() - start,
  });
}
