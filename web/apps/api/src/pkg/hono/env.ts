import type { Env } from "@/pkg/env";
import type { Vault } from "@/pkg/vault";
import type { Ratelimit as UnkeyRatelimiter } from "@unkey/ratelimit";
import type { RBAC } from "@unkey/rbac";
import type { Logger } from "@unkey/worker-logging";
import type { Analytics } from "../analytics";
import type { Cache } from "../cache";
import type { Database } from "../db";
import type { KeyService } from "../keys/service";
import type { Metrics } from "../metrics";
import type { RateLimiter } from "../ratelimit";
import type { UsageLimiter } from "../usagelimit";
export type ServiceContext = {
  rbac: RBAC;
  cache: Cache;
  db: { primary: Database; readonly: Database };
  metrics: Metrics;
  logger: Logger;
  keyService: KeyService;
  analytics: Analytics;
  usageLimiter: UsageLimiter;
  rateLimiter: RateLimiter;
  vault: Vault;
  deprecationRatelimiter?: UnkeyRatelimiter;
};

export type HonoEnv = {
  Bindings: Env;
  Variables: {
    isolateId: string;
    isolateCreatedAt: number;
    requestId: string;
    requestStartedAt: number;
    workspaceId?: string;
    metricsContext: {
      keyId?: string;
      [key: string]: unknown;
    };
    services: ServiceContext;
    /**
     * IP address or region information
     */
    location: string;
    userAgent?: string;
  };
};
