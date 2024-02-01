import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { validator } from "hono/validator";

import { db } from "../db";
import { keys } from "../db/schema";
import { KeyV1, newId, sha256 } from "../keys";

const port = 3000;
const app = new Hono();

const validateCreateKey = validator("json", (value, c) => {
  // TODO implement validation logic
  return value;
});

const validateVerifyKey = validator("json", (value, c) => {
  // TODO implement validation logic
  return value;
});

app.get("/v1/liveness", async (c) => {
  const res = await db.select().from(keys);
  console.log(res);
  return c.text("Running");
});

app.post("/v1/keys/verifyKey", validateVerifyKey, async (c) => {
  const req = c.req.valid("json");
  const hash = await sha256(req.key);
  const dbRes = await db.query.keys.findFirst({
    where: (table, { eq }) => eq(table.hash, hash),
  });
  console.log(dbRes);
  return c.text("Running");
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

  const newKey = await db
    .insert(keys)
    .values({
      id: keyId,
      name: req.name,
      hash,
      start,
      ownerId: req.ownerId,
      createdAt: new Date(),
      meta: req.meta ? JSON.stringify(req.meta) : null,
      remaining: req.remaining,
    })
    .returning();

  return c.json({ newKey });
});

console.log(`Server is running on port ${port}`);

serve({
  fetch: app.fetch,
  port,
});
