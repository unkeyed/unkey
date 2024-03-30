import type { Env } from "@/pkg/env";
import type { RBAC } from "@unkey/rbac";
import type { Analytics } from "../analytics";
import type { SwrCacher } from "../cache/interface";
import type { Database } from "../db";
import type { KeyService } from "../keys/service";
import type { Logger } from "../logging";
import type { Metrics } from "../metrics";
import type { RateLimiter } from "../ratelimit";
import type { UsageLimiter } from "../usagelimit";

export type ServiceContext = {
  rbac: RBAC;
  cache: SwrCacher;
  db: Database;
  metrics: Metrics;
  logger: Logger;
  keyService: KeyService;
  analytics: Analytics;
  usageLimiter: UsageLimiter;
  rateLimiter: RateLimiter;
};

export type HonoEnv = {
  Bindings: Env;
  Variables: {
    requestId: string;
    services: ServiceContext;
    /**
     * IP address or region information
     */
    location: string;
    userAgent?: string;
  };
};
