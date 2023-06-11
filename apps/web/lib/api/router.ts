// import { AuthorizationError, NotFoundError } from "@/lib/errors";
// import { zValidator } from "@hono/zod-validator";
import { zValidator } from "@hono/zod-validator";
import { eq, schema, type Database } from "@unkey/db";

import { Hono } from "hono";
import { HTTPException } from "hono/http-exception";
import { logger } from "hono/logger";
import { z } from "zod";
import { toBase64 } from "./base64";
import { NotFoundError } from "./errors";
import { newId } from "@unkey/id";
import { toBase58 } from "./base58";

export type Bindings = {
  db: Database;
};

export function init<THono extends Hono>(app: THono, { db }: Bindings): THono {
  app.onError((err, c) => {
    console.error(err.message);
    if (err instanceof HTTPException) {
      return err.getResponse();
    }
    return c.json({ error: "Internal Server Error", message: err.message }, { status: 500 });
  });
  app.use("*", logger());
  app.use("*", async (_c, next) => {
    const now = Date.now();
    await next();
    const responseTime = Date.now() - now;
    console.log(`Request took ${responseTime}ms`);
  });

  app.get("/api/v1/liveness", (c) => c.text("ok"));

  /**
   * Create a new API key
   *
   */
  app.post(
    "/api/v1/keys",
    zValidator(
      "json",
      z.object({
        prefix: z.string().optional(),
        name: z.string().optional(),
        apiId: z.string(),
        byteLength: z.number().int().optional().default(32),
        ownerId: z.string().optional(),
        meta: z.record(z.unknown()),
      }),
    ),
    async (c) => {
      const req = c.req.valid("json");

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

      await db
        .insert(schema.keys)
        .values({
          id: keyId,
          apiId: req.apiId,
          name: req.name,
          tenantId: api.tenantId,
          hash,
          ownerId: req.ownerId,
          meta: req.meta,
        })
        .execute();

      return c.jsonT(
        {
          id: keyId,
          key,
        },
        {
          status: 200,
        },
      );
    },
  );

  app.get("/api/v1/keys/:key", async (c) => {
    const buf = await crypto.subtle.digest("sha-256", new TextEncoder().encode(c.req.param("key")));
    const hash = toBase64(buf);
    const key = await db.select().from(schema.keys).where(eq(schema.keys.hash, hash));

    return c.jsonT(
      { key },
      {
        status: 200,
      },
    );
  });

  return app;
}
