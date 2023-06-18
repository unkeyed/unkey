import { Logger } from "./src/logger";
import { init } from "./src/router";
import { Ratelimiter } from "./src/ratelimit";
import { db, type Key } from "./src/db";

import { serve } from "@hono/node-server";
import { Cache } from "./src/cache";
import { env } from "./src/env";
import { Redis } from "ioredis";
import { Tinybird, publishKeyVerification } from "@unkey/tinybird";

const logger = new Logger({ region: process.env.FLY_REGION });
logger.info("Starting app");

const keyCache = new Cache<Key>({ ttlSeconds: 60 });

const redis = new Redis(env.REDIS_URL, { family: 6 });
// await redis.ping().catch((err) => {
//   logger.error("redis ping failed", { error: err.message });
//   process.exit(1);
// })

const tinybird = new Tinybird({ token: env.TINYBIRD_TOKEN });
const ratelimiter = new Ratelimiter({ redis });

const router = init({
  db,
  logger,
  cache: { keys: keyCache },
  ratelimiter,
  tinybird: { publishKeyVerification: publishKeyVerification(tinybird) },
});
const port = parseInt(process.env.PORT ?? "8080");
logger.info("Starting router", { port });

serve({
  fetch: router.fetch,
  port,
});
