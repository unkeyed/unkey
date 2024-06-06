import type { Context, MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

export function benchmarks(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    try {
      c.executionCtx.waitUntil(ping(c, "aws-ga", "https://g.unkey.org/v1/liveness"));
      c.executionCtx.waitUntil(
        ping(c, "koyeb-all", "https://rich-mela-unkey-95820a9c.koyeb.app/v1/liveness"),
      );
      c.executionCtx.waitUntil(ping(c, "aws-route53", "https://r.unkey.org/v1/liveness"));
    } catch (e) {
      c.get("services").logger.warn("benchmark error", {
        error: (e as Error).message,
      });
    }
    return next();
  };
}

async function ping(c: Context, platform: string, url: string): Promise<void> {
  const start = performance.now();
  const res = await fetch(url);
  await res.text();

  c.get("services").metrics.emit({
    metric: "metric.server.latency",
    platform,
    // @ts-expect-error
    colo: c.req.raw?.cf?.colo ?? "unknown",
    // @ts-expect-error
    country: c.req.raw?.cf?.country ?? "unknown",
    // @ts-ignore
    continent: c.req.raw?.cf?.continent ?? "unknown",
    latency: performance.now() - start,
  });
}
