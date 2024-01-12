import { randomBytes } from "crypto";
import { schema } from "@unkey/db";
import baseX from "base-x";

import { connect } from "@planetscale/database";
import { isNotNull } from "drizzle-orm";
import { drizzle } from "drizzle-orm/planetscale-serverless";

const prefixes = {
  role: "role",
};

export function newId(prefix: keyof typeof prefixes): string {
  const buf = randomBytes(16);
  return [
    prefixes[prefix],
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(buf),
  ].join("_");
}

async function main() {
  console.log("RUNNING");
  const db = drizzle(
    connect({
      host: process.env.DATABASE_HOST,
      username: process.env.DATABASE_USERNAME,
      password: process.env.DATABASE_PASSWORD,
    }),
    {
      schema,
    },
  );
  console.log("X");
  const keys = await db.query.keys.findMany({
    where: isNotNull(schema.keys.forWorkspaceId),
    with: { roles: true },
  });
  let i = 0;
  for (const key of keys) {
    console.log("");
    console.log(++i, "/", keys.length, key.id);
    if (key.roles.map((r) => r.role).includes("*")) {
      console.log("SKIPPING");
      continue;
    }

    await db.insert(schema.roles).values({
      id: newId("role"),
      role: "*",
      keyId: key.id,
      workspaceId: key.workspaceId,
    });
  }
}

main();
