import crypto from "node:crypto";
import { Config, connect } from "@planetscale/database";
import { schema } from "@unkey/db";
import { drizzle as drizzleMysql } from "drizzle-orm/mysql2";
import { drizzle as drizzlePS } from "drizzle-orm/planetscale-serverless";

import { z } from "zod";

import baseX from "base-x";
import mysql from "mysql2/promise";

const envSchema = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  DATABASE_NAME: z.string().optional(),
  DATABASE_ADAPTER: z.enum(["planet-scale", "mysql"]).default("planet-scale"),
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
  const {
    DATABASE_ADAPTER,
    DATABASE_HOST,
    DATABASE_PASSWORD,
    DATABASE_USERNAME,
    TENANT_ID,
    DATABASE_NAME,
  } = envSchema.parse(process.env);

  const commonConfig = {
    host: DATABASE_HOST,
    password: DATABASE_PASSWORD,
  };

  const planetScaleConfig: Config = {
    ...commonConfig,
    username: DATABASE_USERNAME,
    // biome-ignore lint/suspicious/noExplicitAny: <explanation>
    fetch: (url: string, init: any) => {
      // biome-ignore lint/suspicious/noExplicitAny: TODO
      (init as any).cache = undefined; // Remove cache header
      return fetch(url, init);
    },
  };

  const mySqlConfig: mysql.ConnectionOptions = {
    ...commonConfig,
    user: DATABASE_USERNAME,
    database: DATABASE_NAME,
  };

  const db =
    DATABASE_ADAPTER === "planet-scale"
      ? drizzlePS(connect(planetScaleConfig), { schema })
      : drizzleMysql(await mysql.createConnection(mySqlConfig), {
          schema,
          mode: "default",
        });

  const workspaceId = newId("ws");

  const workspace = {
    id: workspaceId,
    tenantId: TENANT_ID,
    name: "Unkey",
    internal: true,
    betaFeatures: {},
    features: {},
  };
  await db.insert(schema.workspaces).values(workspace);
  console.log(`Created workspace: ${workspace.name} with id: ${workspace.id}`);

  const keyAuth = {
    id: newId("key_auth"),
    workspaceId: workspace.id,
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
  });
  console.log(`Created API: ${apiId}`);
}

main();
