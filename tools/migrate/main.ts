import { schema, db, eq } from "@unkey/db";
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
  const apis = await db.query.apis.findMany();
  let i = 0;
  for (const api of apis) {
    console.log("");
    console.log(++i, "/", apis.length);
    console.table(api);
    if (api.keyAuthId) {
      console.log("skipping");
      continue;
    }
    const keyAuthId = newId("keyAuth");
    await db.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: api.workspaceId,
    });
    await db
      .update(schema.apis)
      .set({ keyAuthId, authType: "key" })
      .where(eq(schema.apis.id, api.id));
  }
}

main();
