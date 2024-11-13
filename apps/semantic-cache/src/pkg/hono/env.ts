import type { Env } from "@/pkg/env";
import type { Logger } from "@unkey/worker-logging";
import type { Database } from "../db";
import type { Metrics } from "../metrics";

export type ServiceContext = {
  db: Database;
  metrics: Metrics;
  logger: Logger;
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

    tokens?: Promise<number>;
    response?: Promise<string>;
    query?: string;
    vector?: Array<number>;
    cacheHit?: boolean;
    cacheLatency?: number;
    embeddingsLatency?: number;
    vectorizeLatency?: number;
    inferenceLatency?: number;
  };
};
