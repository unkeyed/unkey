import path from "path";
import kleur from "kleur";
import fs from "fs";
import crypto from "node:crypto";
import { connect } from "@planetscale/database";
import { drizzle } from "drizzle-orm/planetscale-serverless";
import { schema } from "../internal/db";

import baseX from "base-x";
import { envSchema, dbSchema } from "../apps/web/lib/env";
import { parseEnv } from "./utils";

const __cwd = process.cwd();

function newId(prefix: "api" | "ws" | "key_auth") {
  const buf = new Uint8Array(16);
  crypto.getRandomValues(buf);
  return [
    prefix,
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(buf),
  ].join("_");
}

const runBootstrap = async () => {
  const web_env = path.resolve(__cwd, "apps/web/.env");

  if (!fs.existsSync(web_env)) {
    console.error(kleur.red(`🚨 Error: .env file does not exist for ${web_env}`));
    process.exit(0);
  }

  const validatedWebEnv = envSchema
    .pick({ TENANT_ID: true })
    .merge(dbSchema)
    .safeParse(parseEnv(web_env));

  if (!validatedWebEnv.success) {
    console.error(validatedWebEnv.error.message);
    process.exit(0);
  }

  const db = drizzle(
    connect({
      host: validatedWebEnv.data.DATABASE_HOST,
      username: validatedWebEnv.data.DATABASE_USERNAME,
      password: validatedWebEnv.data.DATABASE_PASSWORD,
      // biome-ignore lint/suspicious/noExplicitAny: TODO
      fetch: (url: string, init: any) => {
        // biome-ignore lint/suspicious/noExplicitAny: TODO
        (init as any).cache = undefined; // Remove cache header
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );

  const api = await db.query.apis.findFirst().then((r) => {
    return [r?.workspaceId, r?.keyAuthId, r?.id].filter((v) => v != undefined);
  });

  /**
   * In terms of bootstrapping, something is not populated, we should
   * continue.
   */
  if (api.length !== 3) {
    const workspaceId = newId("ws");
    const workspace = {
      id: workspaceId,
      tenantId: validatedWebEnv.data.TENANT_ID,
      name: "Unkey",
      internal: true,
      betaFeatures: {},
      features: {},
    };

    await db.insert(schema.workspaces).values(workspace);

    const keyAuth = {
      id: newId("key_auth"),
      workspaceId: workspace.id,
    };

    await db.insert(schema.keyAuth).values(keyAuth);

    const apiId = newId("api");
    await db.insert(schema.apis).values({
      id: apiId,
      name: "preview",
      workspaceId,
      authType: "key",
      keyAuthId: keyAuth.id,
    });

    console.log({
      workspaceId: workspace.id,
      keyAuthId: keyAuth.id,
      apiId: apiId,
    });
  } else {
    /**
     * Everything is there, just log the variables.
     */

    console.log(kleur.yellow("Bootstrapped vars already exist.\n"));
    console.log({
      workspaceId: api[0],
      keyAuthId: api[1],
      apiId: api[2],
    });
  }
};

runBootstrap().catch((e) => {
  if (e.exitCode === 1) return;
  console.error(e);
  process.exit(1);
});
