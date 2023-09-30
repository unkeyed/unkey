import crypto from "node:crypto";
import { schema } from "@unkey/db";
import { drizzle } from "drizzle-orm/mysql2";
import mysql from "mysql2/promise";
import { z } from "zod";

import baseX from "base-x";
const envSchema = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),

  /**
   * The `organizationId` or `userId` from clerk
   */
  TENANT_ID: z.string(),
});

function newId(prefix: "api" | "ws" | "key_auth") {
  const buf = new Uint8Array(16);
  crypto.getRandomValues(buf);
  return [
    prefix,
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(buf),
  ].join("_");
}

async function main() {
  const env = envSchema.parse(process.env);

  const db = drizzle(
    await mysql.createConnection({
      host: env.DATABASE_HOST,
      user: env.DATABASE_USERNAME,
      password: env.DATABASE_PASSWORD,
    }),
  );

  const workspaceId = newId("ws");

  const workspace = {
    id: workspaceId,
    tenantId: env.TENANT_ID,
    name: "Unkey",
    internal: true,
  };
  await db().insert(schema.workspaces).values(workspace);
  console.log(`Created workspace: ${workspace.name} with id: ${workspace.id}`);

  const keyAuth = {
    id: newId("key_auth"),
    workspaceId: workspace.id,
  };
  await db().insert(schema.keyAuth).values(keyAuth);

  /**
   * Set up an api for production
   */
  const apiId = newId("api");
  await db().insert(schema.apis).values({
    id: apiId,
    name: "preview",
    workspaceId,
    authType: "key",
    keyAuthId: keyAuth.id,
  });
  console.log(`Created API: ${apiId}`);
}

main();
