import { schema, db } from "@unkey/db";
import { z } from "zod";
import crypto from "node:crypto";

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

function newId(prefix: "api" | "ws") {
  const buf = new Uint8Array(16);
  crypto.getRandomValues(buf);
  return [
    prefix,
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(
      buf
    ),
  ].join("_");
}

async function main() {
  const env = envSchema.parse(process.env);

  const workspaceId = newId("ws");

  const workspace = {
    id: workspaceId,
    tenantId: env.TENANT_ID,
    name: "Unkey",
    slug: "unkey",
    internal: true,
  };
  await db.insert(schema.workspaces).values(workspace);
  console.log(`Created workspace: ${workspace.name} with id: ${workspace.id}`);

  /**
   * Set up an api for development
   */
  const devApi = {
    id: newId("api"),
    name: "unkey-dev",
    workspaceId,
  };
  await db.insert(schema.apis).values(devApi);
  console.log(`Created API: ${devApi.name} with id: ${devApi.id}`);

  /**
   * Set up an api for production
   */
  const prodApi = {
    id: newId("api"),
    name: "api.unkey.app",
    workspaceId,
  };
  await db.insert(schema.apis).values(prodApi);
  console.log(`Created API: ${prodApi.name} with id: ${prodApi.id}`);
}

main();
