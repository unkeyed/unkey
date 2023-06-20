import { zValidator } from "@hono/zod-validator";
import { type Database, type Key } from "./db";
import { eq, schema } from "@unkey/db";
import { Hono } from "hono";
import { HTTPException } from "hono/http-exception";
import { logger as honoLogger } from "hono/logger";
import { z } from "zod";
import { AuthorizationError, BadRequestError, NotFoundError } from "./errors";
import { newId } from "@unkey/id";
import { toBase58, toBase64 } from "./encoding";
import { Logger } from "./logger";
import { webcrypto as crypto } from "node:crypto";
import type { Cache } from "./cache";
import { RatelimitResult, Ratelimiter } from "./ratelimit";

import { publishKeyVerification } from "@unkey/tinybird";
import { env } from "./env";
import { Kafka } from "./kafka";

export type Bindings = {
  db: Database;
  logger: Logger;
  ratelimiter: Ratelimiter;
  tinybird: { publishKeyVerification: ReturnType<typeof publishKeyVerification> };
  kafka: Kafka;
  cache: {
    keys: Cache<Key>;
  };
};

export type Variables = {
  requestId: string;
  logger: Logger;
};

export function init({ db, logger, ratelimiter, cache, tinybird, kafka }: Bindings) {
  const app = new Hono<{ Variables: Variables }>();
  app.onError((err, c) => {
    const log = c.get("logger") ?? logger;
    log.error("unhandled error", { error: err.message, stack: err.stack });
    if (err instanceof HTTPException) {
      return err.getResponse();
    }

    return c.json({ error: "Internal Server Error", message: err.message }, { status: 500 });
  });

  // add request id
  app.use("*", async (c, next) => {
    const requestId = newId("request");

    c.set("requestId", requestId);
    c.res.headers.set("x-request-id", requestId);
    await next();
  });

  // add request id
  app.use("*", async (c, next) => {
    const log = logger.with({
      requestId: c.get("requestId"),
      path: c.req.path,
      method: c.req.method,
      edge: c.req.headers.get("FLY_REGION"),
    });
    log.info("incoming request");
    c.set("logger", log);

    await next();
  });

  app.use("*", async (c, next) => {
    const now = performance.now();
    await next();
    const responseTime = performance.now() - now;
    c.get("logger").info("Request duration", { duration: responseTime });
  });

  app.get("/v1/liveness", (c) => c.text("ok"));

  /**
   * Create a new API key
   *
   */
  app.post(
    "/v1/keys",
    zValidator(
      "json",
      z.object({
        prefix: z.string().optional(),
        apiId: z.string(),
        byteLength: z.number().int().optional().default(32),
        ownerId: z.string().optional(),
        meta: z.record(z.unknown()).optional(),
        expires: z.number().int().optional(), // unix timestamp in milliseconds
        ratelimit: z
          .object({
            type: z.enum(["consistent", "fast"]),
            limit: z.number().int().gte(1),
            refillRate: z.number().int().gte(1),
            refillInterval: z.number().int().gte(1),
          })
          .optional(),
      }),
    ),
    async (c) => {
      const log = c.get("logger");
      const req = c.req.valid("json");

      const now = Date.now();

      if (req.expires && req.expires < now) {
        throw new BadRequestError(
          "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?",
        );
      }

      const authorization = c.req.headers.get("authorization");
      if (!authorization) {
        throw new AuthorizationError("Missing Authorization header");
      }
      const token = authorization.replace("Bearer ", "");
      log.info("Got token", { token });
      const unkeyKey = await db.query.keys.findFirst({
        where: eq(
          schema.keys.hash,
          toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(token))),
        ),
      });
      log.info("Found key", unkeyKey);
      if (!unkeyKey) {
        throw new AuthorizationError("Unauthorized");
      }
      log.info("Found key", unkeyKey);
      //  for (const policy of unkeyKey) {
      //    const p = Policy.fromJSON(policy.policy);
      //  }

      if (!unkeyKey.forWorkspaceId) {
        throw new AuthorizationError("Wrong key type");
      }
      log.info("This key belongs to", { workspaceId: unkeyKey.forWorkspaceId });

      const api = await db.query.apis.findFirst({
        where: eq(schema.apis.id, req.apiId),
      });
      if (!api) {
        throw new NotFoundError("API not found");
      }
      if (api.workspaceId !== unkeyKey.forWorkspaceId) {
        throw new AuthorizationError("Unauthorized");
      }

      const buf = new Uint8Array(req.byteLength);
      crypto.getRandomValues(buf);

      let key = toBase58(buf);
      if (req.prefix) {
        key = [req.prefix, key].join("_");
      }

      const hash = toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(key)));

      const keyId = newId("key");

      log.info("Creating key", {
        key,
        keyId,
        apiId: req.apiId,
        workspaceId: unkeyKey.forWorkspaceId,
        hash,
      });

      await db
        .insert(schema.keys)
        .values({
          id: keyId,
          apiId: req.apiId,
          workspaceId: unkeyKey.forWorkspaceId,
          hash,
          ownerId: req.ownerId,
          meta: req.meta,
          start: key.substring(0, (req.prefix?.length ?? 0) + 4),
          createdAt: new Date(),
          expires: req.expires ? new Date(req.expires) : undefined,
          ratelimitType: req.ratelimit?.type,
          ratelimitLimit: req.ratelimit?.limit,
          ratelimitRefillRate: req.ratelimit?.refillRate,
          ratelimitRefillInterval: req.ratelimit?.refillInterval,
        })
        .execute()
        .catch((err) => {
          console.error(err.message);
          throw err;
        });

      return c.jsonT(
        {
          key,
        },
        {
          status: 200,
        },
      );
    },
  );

  /**
   * Delete an API key
   *
   */
  app.delete("/v1/keys/:keyId", async (c) => {
    const log = c.get("logger");
    const keyId = c.req.query("keyId");
    if (!keyId) {
      throw new BadRequestError("Missing keyId");
    }

    const _now = Date.now();

    const authorization = c.req.headers.get("authorization");
    if (!authorization) {
      throw new AuthorizationError("Missing Authorization header");
    }
    const token = authorization.replace("Bearer ", "");
    log.info("Got token", { token });
    const unkeyKey = await db.query.keys.findFirst({
      where: eq(
        schema.keys.hash,
        toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(token))),
      ),
    });
    log.info("Found key", unkeyKey);
    if (!unkeyKey) {
      throw new AuthorizationError("Unauthorized");
    }
    log.info("Found key", unkeyKey);
    //  for (const policy of unkeyKey) {
    //    const p = Policy.fromJSON(policy.policy);
    //  }

    if (!unkeyKey.forWorkspaceId) {
      throw new AuthorizationError("Wrong key type");
    }
    log.info("This key belongs to", { workspaceId: unkeyKey.forWorkspaceId });

    const toBeDeletedKey = await db.query.keys.findFirst({
      where: eq(schema.keys.id, keyId),
    });
    if (!toBeDeletedKey) {
      throw new NotFoundError("Key not found");
    }

    if (toBeDeletedKey.workspaceId !== unkeyKey.forWorkspaceId) {
      throw new AuthorizationError("Unauthorized");
    }

    log.info("Deleting key", {
      keyId,
      workspaceId: unkeyKey.forWorkspaceId,
    });

    await db.delete(schema.keys).where(eq(schema.keys.id, keyId)).execute();

    await kafka.publishKeyDeleted({
      key: {
        id: toBeDeletedKey.id,
        hash: toBeDeletedKey.hash,
      },
    });

    return c.text("OK", {
      status: 202,
    });
  });

  app.post(
    "/v1/keys/verify",

    zValidator(
      "json",
      z.object({
        key: z.string(),
      }),
    ),
    async (c) => {
      const log = c.get("logger");

      const hash = toBase64(
        await crypto.subtle.digest("sha-256", new TextEncoder().encode(c.req.valid("json").key)),
      );

      const beforeCache = performance.now();
      let key: Key | undefined = cache.keys.get(hash);
      log.info("report.cache.key.get", {
        hit: Boolean(key),
        latency: performance.now() - beforeCache,
        key,
      });
      if (!key) {
        const beforeDb = performance.now();
        const found = await db
          .select()
          .from(schema.keys)
          .where(eq(schema.keys.hash, hash))
          .execute();
        log.info("report.database.key.get", {
          hit: found.length > 0,
          latency: performance.now() - beforeDb,
          hash,
        });
        key = found.at(0);
      }

      if (!key) {
        return c.json(
          {
            valid: false,
            error: "Key not found",
          },
          {
            status: 404,
          },
        );
      }

      if (key.expires && key.expires < new Date()) {
        await db.delete(schema.keys).where(eq(schema.keys.id, key.id)).execute();
        return c.json(
          {
            valid: false,
            error: "Key not found",
          },
          {
            status: 404,
          },
        );
      }

      cache.keys.set(hash, key);
      const headers: Record<string, string> = {};
      let ratelimited = false;

      if (key.ratelimitType) {
        const ratelimitRequest = {
          limit: key.ratelimitLimit!,
          refillInterval: key.ratelimitRefillInterval!,
          refillRate: key.ratelimitRefillRate!,
        };

        const beforeRatelimit = performance.now();
        const ratelimit =
          key.ratelimitType === "fast"
            ? ratelimiter.limitLocal(key.id, ratelimitRequest)
            : await ratelimiter.limitGlobal(key.id, ratelimitRequest);
        log.info("report.ratelimit", {
          type: key.ratelimitType,
          pass: ratelimit.pass,
          latency: performance.now() - beforeRatelimit,
        });
        headers["Ratelimit-Limit"] = ratelimit.limit.toString();
        headers["Ratelimit-Remaining"] = ratelimit.remaining.toString();
        headers["Ratelimit-Reset"] = ratelimit.reset.toString();

        if (!ratelimit.pass) {
          ratelimited = true;
        }
      }

      // don't await this, we don't want to block the response
      tinybird
        .publishKeyVerification({
          apiId: key.apiId ?? "",
          workspaceId: key.workspaceId,
          keyId: key.id,
          ratelimited,
          time: Date.now(),
        })
        .then(() => {
          log.info("published to tinybird");
        })
        .catch((err) => {
          log.error("unable to publish to tinybird", { error: err.message });
        });

      return c.json(
        {
          valid: !ratelimited,
          ownerId: key.ownerId ?? undefined,
          meta: key.meta ?? undefined,
        },
        200,
        headers,
      );
    },
  );

  return app;
}
