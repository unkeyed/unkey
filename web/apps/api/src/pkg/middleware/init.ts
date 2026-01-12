import { Analytics } from "@/pkg/analytics";
import { createConnection } from "@/pkg/db";

import { KeyService } from "@/pkg/keys/service";
import { DurableRateLimiter } from "@/pkg/ratelimit";
import { DurableUsageLimiter, NoopUsageLimiter } from "@/pkg/usagelimit";
import { RBAC } from "@unkey/rbac";
import { ConsoleLogger } from "@unkey/worker-logging";

import { newId } from "@unkey/id";
import { Ratelimit as UnkeyRatelimiter } from "@unkey/ratelimit";
import type { MiddlewareHandler } from "hono";
import { initCache } from "../cache";
import type { HonoEnv } from "../hono/env";
import { type Metrics, NoopMetrics } from "../metrics";
import { LogdrainMetrics } from "../metrics/logdrain";
import { Vault } from "../vault";

/**
 * These maps persist between worker executions and are used for caching
 */
const rlMap = new Map();

/**
 * workerId and coldStartAt are used to track the lifetime of the worker
 * and are set once when the worker is first initialized.
 *
 * subsequent requests will use the same workerId and coldStartAt
 */
let isolateId: string | undefined = undefined;
let isolateCreatedAt: number | undefined = undefined;
/**
 * Create a Hono middleware that initializes request and worker context and registers services.
 *
 * Initializes worker-level identifiers when first run, sets per-request identifiers and the
 * `Unkey-Request-Id` response header, and constructs/attaches database, metrics, rate limiting,
 * analytics, cache, RBAC, key service, vault, logger, and usage limiter instances to the context
 * under the `services` key.
 *
 * @returns A middleware handler that prepares context for downstream handlers and sets the `Unkey-Request-Id` response header.
 */
export function init(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    if (!isolateId) {
      isolateId = crypto.randomUUID();
    }
    if (!isolateCreatedAt) {
      isolateCreatedAt = Date.now();
    }
    c.set("isolateId", isolateId);
    c.set("isolateCreatedAt", isolateCreatedAt);
    const requestId = newId("request");
    c.set("requestId", requestId);

    c.set("requestStartedAt", Date.now());

    c.res.headers.set("Unkey-Request-Id", requestId);

    const deprecationRatelimiter = c.env.UNKEY_ROOT_KEY
      ? new UnkeyRatelimiter({
          rootKey: c.env.UNKEY_ROOT_KEY,
          namespace: "v1_deprecation",
          limit: 9,
          duration: 60_000,
          timeout: {
            ms: 5000,
            fallback: {
              limit: 10,
              remaining: 10,
              success: true,
              reset: Date.now() + 60_000,
            },
          },
        })
      : undefined;

    const logger = new ConsoleLogger({
      requestId,
      application: "api",
      environment: c.env.ENVIRONMENT,
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
      ? new LogdrainMetrics({
          requestId,
          environment: c.env.ENVIRONMENT,
          isolateId,
        })
      : new NoopMetrics();

    const usageLimiter = c.env.DO_USAGELIMIT
      ? new DurableUsageLimiter({
          requestId,
          namespace: c.env.DO_USAGELIMIT,
          logger,
          metrics,
        })
      : new NoopUsageLimiter();

    const analytics = new Analytics({
      clickhouseUrl: c.env.CLICKHOUSE_URL,
      clickhouseProxyUrl: c.env.CLICKHOUSE_PROXY_URL,
      clickhouseProxyToken: c.env.CLICKHOUSE_PROXY_TOKEN,
    });
    const rateLimiter = new DurableRateLimiter({
      namespace: c.env.DO_RATELIMIT,
      cache: rlMap,
      logger,
      metrics,
    });

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
    });

    const vault = new Vault(c.env.AGENT_URL, c.env.AGENT_TOKEN, metrics);

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
      deprecationRatelimiter,
    });

    await next();
  };
}