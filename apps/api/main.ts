import { Logger } from "./src/logger";
import { init } from "./src/router";
import { Ratelimiter } from "./src/ratelimit";
import { db, type Key } from "./src/db";

import { serve } from "@hono/node-server";
import { Cache } from "./src/cache";
import { env } from "./src/env";
import { Redis } from "ioredis";
import { Tinybird, publishKeyVerification } from "@unkey/tinybird";
import { Kafka } from "./src/kafka";
async function main() {
  const logger = new Logger({ region: env.FLY_REGION, allocationID: env.FLY_ALLOC_ID });
  logger.info("Starting app");

  const keyCache = new Cache<Key>({ ttlSeconds: 10 });

  const kafka = new Kafka({ logger });
  kafka.onKeyDeleted(({ key }) => {
    logger.info("deleting key from cache", key);
    keyCache.delete(key.hash);
  });

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
}

main();
