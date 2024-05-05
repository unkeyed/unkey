import { exec } from "node:child_process";
import path from "node:path";
import { type Api, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { drizzle } from "drizzle-orm/mysql2";
import mysql from "mysql2/promise";
import { task } from "./util";

export async function prepareDatabase(): Promise<{
  workspace: { id: string };
  api: { id: string };
  webhooksApi: { id: string };
}> {
  const db = await connectDatabase();
  await task("migrating tabels", async (s) => {
    const cwd = path.join(__dirname, "../../../internal/db");

    await new Promise((resolve, reject) => {
      const p = exec("pnpm drizzle-kit push:mysql", {
        env: {
          DRIZZLE_DATABASE_URL: "mysql://unkey:password@localhost:3306/unkey",
          ...process.env,
        },

        cwd,
      });
      p.on("exit", (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(code);
        }
      });
    });
    s.stop("table migration complete");
  });

  // fs.unlinkSync(migrationsFolder);

  return await task("Seeding database", async (s) => {
    const rootWorkspaceId = "ws_rootworkspace";

    const existingRootWorkspace = await db.query.workspaces.findFirst({
      where: (table, { eq }) => eq(table.id, rootWorkspaceId),
      with: {
        apis: true,
      },
    });
    if (existingRootWorkspace) {
      s.stop("reusing existing root workspace");
      return {
        workspace: existingRootWorkspace,
        api: existingRootWorkspace.apis.find((api) => api.name === "Unkey")!,
        webhooksApi: existingRootWorkspace.apis.find((api) => api.name === "Unkey Webhooks")!,
      };
    }

    const workspace = {
      id: rootWorkspaceId,
      tenantId: "user_REPLACE_ME_LATER",
      name: "Unkey",
      internal: true,
      betaFeatures: {},
      features: {},
      createdAt: new Date(),
    };
    await db.insert(schema.workspaces).values(workspace);
    s.message("Created root workspace");

    const webhookKeyAuth = {
      id: newId("keyAuth"),
      workspaceId: workspace.id,
      createdAt: new Date(),
    };

    await db.insert(schema.keyAuth).values(webhookKeyAuth);
    s.message("Created webhook key space");

    /**
     * Set up an api for webhook keys
     */
    const webhooksApi = {
      id: newId("api"),
      name: "Unkey Webhooks",
      workspaceId: workspace.id,
      authType: "key",
      keyAuthId: webhookKeyAuth.id,
      createdAt: new Date(),
      deletedAt: null,
      ipWhitelist: null,
    } satisfies Api;
    await db.insert(schema.apis).values(webhooksApi);
    s.message("Created webhook api");

    const keyAuth = {
      id: newId("keyAuth"),
      workspaceId: workspace.id,
      createdAt: new Date(),
    };

    await db.insert(schema.keyAuth).values(keyAuth);
    s.message("Created root keyspace");

    /**
     * Set up an api for production
     */
    const api = {
      id: newId("api"),
      name: "Unkey",
      workspaceId: workspace.id,
      authType: "key",
      keyAuthId: keyAuth.id,
      createdAt: new Date(),
      deletedAt: null,
      ipWhitelist: null,
    } satisfies Api;
    await db.insert(schema.apis).values(api);
    s.message("Created root api");

    s.stop("seed done");
    return { workspace, api, webhooksApi };
  });
}

async function connectDatabase() {
  return await task("Connecting to database", async (s) => {
    let err: Error | undefined = undefined;
    for (let i = 1; i <= 10; i++) {
      try {
        const conn = await mysql.createConnection("mysql://unkey:password@localhost:3306/unkey");

        s.message("pinging database");
        await conn.ping();
        s.stop("connected to database");
        return drizzle(conn, { schema, mode: "default" });
      } catch (e) {
        err = e as Error;
        await new Promise((r) => setTimeout(r, 1000 * i));
      }
    }

    throw err;
  });
}
