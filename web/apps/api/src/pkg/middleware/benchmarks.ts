import type { MiddlewareHandler } from "hono";
import type { Context } from "../hono/app";
import type { HonoEnv } from "../hono/env";
import { instrumentedFetch } from "../util/instrument-fetch";
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
  const res = await instrumentedFetch(c)(url);
  if (!res.ok || res.status !== 200) {
    c.get("services").logger.warn("ping to server failed", {
      status: res.status,
      body: await res.text(),
      url,
    });
  }

  c.get("services").metrics.emit({
    metric: "metric.server.latency",
    platform,
    status: res.status,
    // @ts-expect-error
    country: c.req.raw?.cf?.country ?? "unknown",
    // @ts-expect-error

    continent: c.req.raw?.cf?.continent ?? "unknown",
    // @ts-ignore
    colo: c.req.raw?.cf?.colo ?? "unknown",
    latency: performance.now() - start,
  });
}
