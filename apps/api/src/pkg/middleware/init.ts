import { Analytics } from "@/pkg/analytics";
import { createConnection } from "@/pkg/db";

import { KeyService } from "@/pkg/keys/service";
import { AgentRatelimiter } from "@/pkg/ratelimit";
import { DurableUsageLimiter, NoopUsageLimiter } from "@/pkg/usagelimit";
import { RBAC } from "@unkey/rbac";
import { ConsoleLogger } from "@unkey/worker-logging";

import { newId } from "@unkey/id";
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
 * Initialize all services.
 *
 * Call this once before any hono handlers run.
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
      clickhouse: c.env.CLICKHOUSE_URL
        ? {
            url: c.env.CLICKHOUSE_URL,
          }
        : undefined,
    });
    const rateLimiter = new AgentRatelimiter({
      agent: { url: c.env.AGENT_URL, token: c.env.AGENT_TOKEN },
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
    });

    await next();
  };
}
