import type { Metric } from "@unkey/metrics";
import type { MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

type DiscriminateMetric<T, M = Metric> = M extends { metric: T } ? M : never;

export function metrics(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    const { logger, metrics, analytics } = c.get("services");
    // logger.info("request", {
    //   method: c.req.method,
    //   path: c.req.path,
    // });
    const start = performance.now();
    const m = {
      metric: "metric.http.request",
      path: c.req.path,
      method: c.req.method,
      // @ts-ignore - this is a bug in the types
      continent: c.req.raw?.cf?.continent,
      // @ts-ignore - this is a bug in the types
      country: c.req.raw?.cf?.country,
      // @ts-ignore - this is a bug in the types
      colo: c.req.raw?.cf?.colo,
      // @ts-ignore - this is a bug in the types
      city: c.req.raw?.cf?.city,
      userAgent: c.req.header("user-agent"),
      fromAgent: c.req.header("Unkey-Redirect"),
    } as DiscriminateMetric<"metric.http.request">;
    try {
      m.requestId = c.get("requestId");

      const telemetry = {
        runtime: c.req.header("Unkey-Telemetry-Runtime"),
        platform: c.req.header("Unkey-Telemetry-Platform"),
        versions: c.req.header("Unkey-Telemetry-SDK")?.split(","),
      };
      if (telemetry.runtime || telemetry.platform || telemetry.versions) {
        c.executionCtx.waitUntil(
          analytics
            .ingestSdkTelemetry({
              runtime: telemetry.runtime || "unknown",
              platform: telemetry.platform || "unknown",
              versions: telemetry.versions || [],
              requestId: m.requestId,
              time: Date.now(),
            })
            .catch((err) => {
              logger.error("Error ingesting SDK telemetry", {
                method: c.req.method,
                path: c.req.path,
                error: err.message,
              });
            }),
        );
      }

      await next();
    } catch (e) {
      m.error = (e as Error).message;
      c.get("services").logger.error("request", {
        method: c.req.method,
        path: c.req.path,
        error: e,
      });
      throw e;
    } finally {
      m.status = c.res.status;
      m.serviceLatency = performance.now() - start;
      c.res.headers.append("Unkey-Latency", `service=${m.serviceLatency}ms`);
      c.res.headers.append("Unkey-Version", c.env.VERSION);
      metrics.emit(m);
      c.executionCtx.waitUntil(Promise.all([metrics.flush(), logger.flush()]));
    }
  };
}
