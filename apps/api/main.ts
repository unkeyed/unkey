import { Logger } from "./src/logger";
import { init } from "./src/router";
import { Ratelimiter } from "./src/ratelimit";
import { db, type Key } from "./src/db";

import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { Cache } from "./src/cache";
const logger = new Logger();

const keyCache = new Cache<Key>({ ttlSeconds: 10 });
const ratelimiter = new Ratelimiter();

const router = init(new Hono(), { db, logger, cache: { keys: keyCache }, ratelimiter });
const port = parseInt(process.env.PORT ?? "8080");
logger.info("Starting", { port });

serve({
  fetch: router.fetch,
  port,
});
