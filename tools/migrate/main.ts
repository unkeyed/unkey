import { schema, db, eq, isNull } from "@unkey/db";
import { randomBytes } from "node:crypto";
import baseX from "base-x";

const prefixes = {
  key: "key",
  policy: "pol",
  api: "api",
  request: "req",
  workspace: "ws",
  keyAuth: "key_auth",
} as const;

export function newId(prefix: keyof typeof prefixes): string {
  const buf = randomBytes(16);
  return [
    prefixes[prefix],
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(buf),
  ].join("_");
}

async function main() {
  const keys = await db.query.keys.findMany({ where: isNull(schema.keys.keyAuthId) });
  let i = 0;
  for (const key of keys) {
    console.log("");
    console.log(++i, "/", keys.length);
    console.table(key);
    if (!key.apiId) {
      console.error("key also doesn't have old apiId", key);
      continue;
    }

    const api = await db.query.apis.findFirst({ where: eq(schema.apis.id, key.apiId) });
    if (!api) {
      console.error("api doesn't exist", key);
      continue;
    }
    if (!api.keyAuthId) {
      console.error("api doesn't have keyAuth", key);
      continue;
    }

    console.log("updating key %s with %s", key.id, api.keyAuthId);
    await db
      .update(schema.keys)
      .set({ keyAuthId: api.keyAuthId })
      .where(eq(schema.keys.id, key.id));

    // const keyAuthId = newId("keyAuth");
    // await db.insert(schema.keyAuth).values({
    //   id: keyAuthId,
    //   workspaceId: api.workspaceId,
    // });
    // await db
    //   .update(schema.apis)
    //   .set({ keyAuthId, authType: "key" })
    //   .where(eq(schema.apis.id, api.id));

    // await db.update(schema.keys).set({ keyAuthId }).where(eq(schema.keys.apiId, api.id));
  }
}

main();
