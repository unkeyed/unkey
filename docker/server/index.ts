import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { validator } from "hono/validator";

import { eq } from "drizzle-orm";
import { db } from "../db";
import { keys } from "../db/schema";
import { KeyV1, newId, sha256 } from "../keys";

const port = 3000;
const app = new Hono();

const validateCreateKey = validator("json", (value, c) => {
  const authorization = c.req.header("Authorization");
  if (!authorization) {
    throw new Error("Unauthorized: key required");
  }
  return value;
});

const validateVerifyKey = validator("json", (value, c) => {
  if (!value.key) {
    throw new Error("Unauthorized: key required.");
  }
  return value;
});

app.get("/v1/liveness", async (c) => {
  return c.text("Running");
});

app.post("/v1/keys/verifyKey", validateVerifyKey, async (c) => {
  const req = c.req.valid("json");
  const hash = await sha256(req.key);
  try {
    await db.transaction(async (tx) => {
      const key = await tx.query.keys.findFirst({
        where: (table, { eq }) => eq(table.hash, hash),
      });

      if (key?.remaining) {
        await tx
          .update(keys)
          .set({ remaining: key.remaining - 1 })
          .where(eq(keys.id, key.id));
      }
    });

    const keyAfterUpdate = await db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.hash, hash),
    });

    if (!keyAfterUpdate) {
      return c.json({ status: 404, message: "Key not found" });
    }

    return c.json({
      keyId: keyAfterUpdate.id,
      meta: keyAfterUpdate.meta ?? undefined,
      name: keyAfterUpdate.name ?? undefined,
      ownerId: keyAfterUpdate.ownerId ?? undefined,
      remaining: keyAfterUpdate.remaining ?? undefined,
    });
  } catch (_error) {
    return c.json({ status: 500, message: "Internal server error" });
  }
});

app.post("/v1/keys/createKey", validateCreateKey, async (c) => {
  const req = c.req.valid("json");
  const key = new KeyV1({
    byteLength: req.byteLength ?? 16,
    prefix: req.prefix,
  }).toString();
  const start = key.slice(0, (req.prefix?.length ?? 0) + 5);
  const keyId = newId("key");
  const hash = await sha256(key.toString());

  await db.insert(keys).values({
    id: keyId,
    name: req.name,
    hash,
    start,
    ownerId: req.ownerId,
    createdAt: new Date(),
    meta: req.meta ? JSON.stringify(req.meta) : null,
    remaining: req.remaining,
  });

  return c.json({ key, id: keyId });
});

console.log(`Server is running on port ${port}`);

serve({
  fetch: app.fetch,
  port,
});
