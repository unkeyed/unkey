import { zValidator } from "@hono/zod-validator";
import { type Database, type Key } from "./db";
import { eq, schema } from "@unkey/db";
import { Hono } from "hono";
import { HTTPException } from "hono/http-exception";
import { logger as honoLogger } from "hono/logger";
import { z } from "zod";
import { AuthorizationError, NotFoundError } from "./errors";
import { newId } from "@unkey/id";
import { toBase58, toBase64 } from "./encoding";
import { Logger } from "./logger";
import { webcrypto as crypto } from "node:crypto";
import type { Cache } from "./cache";
import { RatelimitResult, Ratelimiter } from "./ratelimit";

export type Bindings = {
  db: Database;
  logger: Logger;
  ratelimiter: Ratelimiter;
  cache: {
    keys: Cache<Key>;
  };
};

export function init<THono extends Hono>(app: THono, { db, logger, ratelimiter }: Bindings): THono {
  app.onError((err, c) => {
    logger.error(err.message);
    if (err instanceof HTTPException) {
      return err.getResponse();
    }
    return c.json({ error: "Internal Server Error", message: err.message }, { status: 500 });
  });
  app.use("*", honoLogger());
  app.use("*", async (_c, next) => {
    const now = Date.now();
    await next();
    const responseTime = Date.now() - now;
    logger.info(`Request took ${responseTime}ms`);
  });

  app.get("/v1/liveness", (c) => c.text("ok"));

  app.get("/v1/tenants", async (c) => {
    const tenants = await db.query.tenants.findMany();
    return c.json({ tenants });
  });
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
      const authorization = c.req.headers.get("authorization");
      if (!authorization) {
        throw new AuthorizationError("Missing Authorization header");
      }
      const token = authorization.replace("Bearer ", "");
      const unkeyKey = await db.query.keys.findFirst({
        where: eq(
          schema.keys.hash,
          toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(token))),
        ),
      });

      if (!unkeyKey) {
        throw new AuthorizationError("Unauthorized");
      }

      const req = c.req.valid("json");
      if (unkeyKey.apiId !== req.apiId) {
        throw new AuthorizationError("Unauthorized to access this api");
      }

      const api = await db.query.apis.findFirst({
        where: eq(schema.apis.id, req.apiId),
        columns: { tenantId: true },
      });
      if (!api) {
        throw new NotFoundError("This api does not exist");
      }

      const buf = new Uint8Array(req.byteLength);
      crypto.getRandomValues(buf);

      let key = toBase58(buf);
      if (req.prefix) {
        key = [req.prefix, key].join("_");
      }

      const hash = toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(key)));

      const keyId = newId("key");

      logger.info("Creating key", { key, keyId, apiId: req.apiId, tenantId: api.tenantId });

      await db
        .insert(schema.keys)
        .values({
          id: keyId,
          apiId: req.apiId,
          tenantId: api.tenantId,
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

  app.post(
    "/v1/keys/verify",

    zValidator(
      "json",
      z.object({
        key: z.string(),
      }),
    ),
    async (c) => {
      const buf = await crypto.subtle.digest(
        "sha-256",
        new TextEncoder().encode(c.req.valid("json").key),
      );
      const hash = toBase64(buf);
      const found = await db.select().from(schema.keys).where(eq(schema.keys.hash, hash)).execute();
      const key = found.at(0);

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

      const headers: Record<string, string> = {};

      if (key.ratelimitType) {
        const ratelimitRequest = {
          limit: key.ratelimitLimit,
          refillInterval: key.ratelimitRefillInterval,
          refillRate: key.ratelimitRefillRate,
        };

        const ratelimit =
          key.ratelimitType === "fast"
            ? ratelimiter.limitLocal(key.id, ratelimitRequest)
            : await ratelimiter.limitGlobal(key.id, ratelimitRequest);

        headers["Ratelimit-Limit"] = ratelimit.limit.toString();
        headers["Ratelimit-Remaining"] = ratelimit.remaining.toString();
        headers["Ratelimit-Reset"] = ratelimit.reset.toString();

        if (!ratelimit.pass) {
          return c.json(
            {
              valid: false,
              error: "Ratelimit exceeded",
            },
            429,
            headers,
          );
        }
      }

      return c.json(
        {
          valid: true,
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
