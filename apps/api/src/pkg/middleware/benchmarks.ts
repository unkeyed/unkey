import type { MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

export function benchmarks(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    try {
      const { metrics } = c.get("services");
      const start = performance.now();
      const p = fetch(
        "https://rich-mela-unkey-95820a9c.koyeb.app/ratelimit.v1.RatelimitService/Ratelimit",
        {
          method: "POST",
          body: JSON.stringify({
            identifier: "chronark",
            limit: 10,
            duration: 10000,
          }),
        },
      ).then(async (res) => {
        const json = (await res.json()) as { success?: boolean };
        metrics.emit({
          metric: "metric.ratelimit.latency",
          platform: "koyeb",
          latency: performance.now() - start,
          success: json.success ?? false,
        });
      });

      c.executionCtx.waitUntil(p);
    } catch {}
    return next();
  };
}
