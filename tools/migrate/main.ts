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
  await db.delete(schema.keyAuth);
  const apis = await db.query.apis.findMany();
  let i = 0;
  for (const api of apis) {
    console.log("");
    console.log(i++, "/", apis.length);
    console.table(api);
    await db.update(schema.apis).set({ authType: "key" }).where(eq(schema.apis.id, api.id));
    if (api.keyAuthId !== "key_auth_APcipZWD9qb3ZAMumtDzUA") {
      console.log("skipping");
      continue;
    }
    const keyAuthId = newId("keyAuth");
    await db.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: api.workspaceId,
    });
    await db.update(schema.apis).set({ keyAuthId }).where(eq(schema.apis.id, api.id));
  }
}

main();
