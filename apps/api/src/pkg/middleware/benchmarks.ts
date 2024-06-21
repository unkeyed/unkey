import type { Context, MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

export function benchmarks(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    try {
      c.executionCtx.waitUntil(ping(c, "aws", "https://aws.unkey.cloud/v1/liveness"));
      c.executionCtx.waitUntil(ping(c, "fly", "https://unkey-agent-dev.fly.dev/v1/liveness"));
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
    country: c.req.raw?.cf?.country ?? "unknown",
    // @ts-ignore
    colo: c.req.raw?.cf?.colo ?? "unknown",
    latency: performance.now() - start,
  });
}
