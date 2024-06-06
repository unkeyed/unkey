import type { Env } from "@/pkg/env";
import type { Logger } from "@unkey/worker-logging";
import type { Analytics } from "../analytics";
import type { Cache } from "../cache";
import type { Database } from "../db";
import type { Metrics } from "../metrics";

export type ServiceContext = {
  cache: Cache;
  db: Database;
  metrics: Metrics;
  logger: Logger;
  analytics: Analytics;
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

    tokens?: number;
    response?: string;
    query?: string;
    vector?: Array<number>;
    cacheHit?: boolean;
  };
};
