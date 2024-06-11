import { createConnection } from "@/pkg/db";
import { ConsoleLogger } from "@unkey/worker-logging";

import { newId } from "@unkey/id";
import type { MiddlewareHandler } from "hono";
import { Analytics } from "../analytics";
import { initCache } from "../cache";
import type { HonoEnv } from "../hono/env";
import { type Metrics, NoopMetrics } from "../metrics";
import { LogdrainMetrics } from "../metrics/logdrain";

/**
 * Initialize all services.
 *
 * Call this once before any hono handlers run.
 */
export function init(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    const requestId = newId("request");
    c.res.headers.set("Unkey-Request-Id", requestId);
    c.set("requestId", requestId);

    const db = createConnection(c.env);

    const metrics: Metrics = c.env.EMIT_METRICS_LOGS
      ? new LogdrainMetrics({ requestId, environment: c.env.ENVIRONMENT })
      : new NoopMetrics();

    const logger = new ConsoleLogger({
      requestId,
      application: "semantic-cache",
      environment: c.env.ENVIRONMENT,
    });

    const tinybirdProxy =
      c.env.TINYBIRD_PROXY_URL && c.env.TINYBIRD_PROXY_TOKEN
        ? {
            url: c.env.TINYBIRD_PROXY_URL,
            token: c.env.TINYBIRD_PROXY_TOKEN,
          }
        : undefined;

    const analytics = new Analytics({
      tinybirdProxy,
      tinybirdToken: c.env.TINYBIRD_TOKEN,
    });

    const cache = initCache(c, metrics);

    c.set("services", {
      db,
      metrics,
      logger,
      analytics,
      cache,
    });

    await next();
  };
}
