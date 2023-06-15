import { Logger } from "./src/logger";
import { init } from "./src/router";
import { Ratelimiter } from "./src/ratelimit";
import { db, type Key } from "./src/db";

import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { Cache } from "./src/cache";
import { env } from "./src/env";

const logger = new Logger();

const keyCache = new Cache<Key>({ ttlSeconds: 60 });
const ratelimiter = new Ratelimiter(env.REDIS_URL);

const router = init(new Hono(), { db, logger, cache: { keys: keyCache }, ratelimiter });
const port = parseInt(process.env.PORT ?? "8080");
logger.info("Starting", { port });

serve({
  fetch: router.fetch,
  port,
});
