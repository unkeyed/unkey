import crypto from "node:crypto";
import { connect } from "@planetscale/database";
import { schema } from "@unkey/db";
import { drizzle } from "drizzle-orm/planetscale-serverless";
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
    connect({
      host: env.DATABASE_HOST,
      username: env.DATABASE_USERNAME,
      password: env.DATABASE_PASSWORD,

      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );

  const workspaceId = newId("ws");

  const workspace = {
    id: workspaceId,
    tenantId: env.TENANT_ID,
    name: "Unkey",
    internal: true,
    betaFeatures: {},
    features: {},
    createdAt: new Date(),
  };
  await db.insert(schema.workspaces).values(workspace);

  const keyAuth = {
    id: newId("key_auth"),
    workspaceId: workspace.id,
    createdAt: new Date(),
  };

  await db.insert(schema.keyAuth).values(keyAuth);

  /**
   * Set up an api for production
   */
  const apiId = newId("api");
  await db.insert(schema.apis).values({
    id: apiId,
    name: "preview",
    workspaceId,
    authType: "key",
    keyAuthId: keyAuth.id,
    createdAt: new Date(),
  });

  console.log("Add these to /apps/agent/.env and /apps/web/.env");
  console.log(`
UNKEY_WORKSPACE_ID="${workspaceId}"
UNKEY_API_ID="${apiId}"
UNKEY_KEY_AUTH_ID="${keyAuth.id}"
    `);
}

main();
