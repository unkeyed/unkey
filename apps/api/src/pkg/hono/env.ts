import type { Env } from "@/pkg/env";
import { RBAC } from "@unkey/rbac";
import { Analytics } from "../analytics";
import { SwrCacher } from "../cache/interface";
import { Database } from "../db";
import { KeyService } from "../keys/service";
import { Logger } from "../logging";
import { Metrics } from "../metrics";
import { RateLimiter } from "../ratelimit";
import { UsageLimiter } from "../usagelimit";

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
