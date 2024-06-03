import { Analytics } from "@/pkg/analytics";
import { createConnection } from "@/pkg/db";
import { KeyService } from "@/pkg/keys/service";
import { DurableRateLimiter, NoopRateLimiter } from "@/pkg/ratelimit";
import { DurableUsageLimiter, NoopUsageLimiter } from "@/pkg/usagelimit";
import { RBAC } from "@unkey/rbac";
import { ConsoleLogger } from "@unkey/worker-logging";

import { newId } from "@unkey/id";
import type { MiddlewareHandler } from "hono";
import { initCache } from "../cache";
import type { HonoEnv } from "../hono/env";
import { type Metrics, NoopMetrics } from "../metrics";
import { LogdrainMetrics } from "../metrics/logdrain";
import { connectVault } from "../vault";

/**
 * These maps persist between worker executions and are used for caching
 */
const rlMap = new Map();

/**
 * Initialize all services.
 *
 * Call this once before any hono handlers run.
 */
export function init(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    const requestId = newId("request");
    c.set("requestId", requestId);
    c.res.headers.set("Unkey-Request-Id", requestId);

    const logger = new ConsoleLogger({
      requestId,
      defaultFields: { environment: c.env.ENVIRONMENT },
    });
    const primary = createConnection({
      host: c.env.DATABASE_HOST,
      username: c.env.DATABASE_USERNAME,
      password: c.env.DATABASE_PASSWORD,
      retry: 3,
      logger,
    });

    const readonly =
      c.env.DATABASE_HOST_READONLY &&
      c.env.DATABASE_USERNAME_READONLY &&
      c.env.DATABASE_PASSWORD_READONLY
        ? createConnection({
            host: c.env.DATABASE_HOST_READONLY,
            username: c.env.DATABASE_USERNAME_READONLY,
            password: c.env.DATABASE_PASSWORD_READONLY,
            retry: 3,
            logger,
          })
        : primary;

    const db = { primary, readonly };

    const metrics: Metrics = c.env.EMIT_METRICS_LOGS
      ? new LogdrainMetrics({ requestId })
      : new NoopMetrics();

    const usageLimiter = c.env.DO_USAGELIMIT
      ? new DurableUsageLimiter({
          requestId,
          namespace: c.env.DO_USAGELIMIT,
          logger,
          metrics,
        })
      : new NoopUsageLimiter();

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
    const rateLimiter = c.env.DO_RATELIMIT
      ? new DurableRateLimiter({
          cache: rlMap,
          namespace: c.env.DO_RATELIMIT,
          logger,
          metrics,
        })
      : new NoopRateLimiter();

    const cache = initCache(c, metrics);

    const rbac = new RBAC();
    const keyService = new KeyService({
      rbac,
      cache,
      logger,
      db,
      metrics,
      rateLimiter,
      usageLimiter,
      analytics,
    });

    const vault = connectVault(c.env, metrics);

    c.set("services", {
      vault,
      rbac,
      db,
      metrics,
      logger,
      usageLimiter,
      rateLimiter,
      analytics,
      cache,
      keyService,
    });

    await next();
  };
}
