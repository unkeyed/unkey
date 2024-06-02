import type { MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";
import type { Metrics } from "../metrics";

export function benchmarks(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    try {
      const { metrics } = c.get("services");

      c.executionCtx.waitUntil(testAWS(metrics));
      c.executionCtx.waitUntil(testKoyeb(metrics));
    } catch {}
    return next();
  };
}

async function testAWS(metrics: Metrics): Promise<void> {
  const start = performance.now();
  const res = await fetch("https://api-56140r9a.fctl.app/v1/liveness", {
    method: "POST",
  });
  await res.text();

  metrics.emit({
    metric: "metric.server.latency",
    platform: "aws",
    // @ts-expect-error
    country: c.req.raw?.cf?.country ?? "unknown",
    latency: performance.now() - start,
  });
}

async function testKoyeb(metrics: Metrics): Promise<void> {
  const start = performance.now();
  const res = await fetch(
    "https://rich-mela-unkey-95820a9c.koyeb.app/ratelimit.v1.RatelimitService/Ratelimit",
    {
      method: "POST",
      body: JSON.stringify({
        identifier: "chronark",
        limit: 10,
        duration: 10000,
      }),
    },
  );
  await res.text();
  metrics.emit({
    metric: "metric.server.latency",
    platform: "koyeb",
    // @ts-expect-error
    country: c.req.raw?.cf?.country ?? "unknown",
    latency: performance.now() - start,
  });
}
