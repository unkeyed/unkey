import { exec } from "node:child_process";
import path from "node:path";
import { mysqlDrizzle, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";
import { task } from "./util";

const ROW_IDS = {
  rootWorkspace: "ws_local_root",
  rootKeySpace: "ks_local_root_keys",
  rootApi: "api_local_root_keys",
  webhookKeySpace: "ks_local_webhook_keys",
  webhookApi: "api_local_webhook_keys",
};

export async function prepareDatabase(url?: string): Promise<{
  workspace: { id: string };
  api: { id: string };
  webhooksApi: { id: string };
}> {
  const db = await connectDatabase();
  await task("migrating tables", async (s) => {
    const cwd = path.join(__dirname, "../../../internal/db");

    await new Promise((resolve, reject) => {
      let stdoutData = "";
      const p = exec("pnpm drizzle-kit push", {
        env: {
          DRIZZLE_DATABASE_URL: url ?? "mysql://unkey:password@localhost:3306/unkey",
          ...process.env,
        },
        cwd,
      });

      p.stdout?.on("data", (data) => {
        stdoutData += data;
      });

      p.on("exit", (code) => {
        if (code === 0) {
          if (stdoutData.includes("Error")) {
            reject(new Error(`Migration failed with MySQL error:\n${stdoutData}`));
          }

          resolve(code);
        } else {
          reject(code);
        }
      });
    });
    s.stop("table migration complete");
  });

  return await task("Seeding database", async (s) => {
    // root workspace
    await db
      .insert(schema.workspaces)
      .values({
        id: ROW_IDS.rootWorkspace,
        tenantId: "user_REPLACE_ME",
        name: "Unkey",
        createdAt: new Date(),
        betaFeatures: {},
        features: {},
      })
      .onDuplicateKeyUpdate({ set: { createdAt: new Date() } });

    s.message("Created root workspace");

    await db
      .insert(schema.auditLogBucket)
      .values({
        id: newId("auditLogBucket"),
        workspaceId: ROW_IDS.rootWorkspace,
        name: "unkey_mutations",
        deleteProtection: true,
      })
      .onDuplicateKeyUpdate({ set: { createdAt: Date.now() } });
    s.message("Created audit log bucket");

    await db
      .insert(schema.keyAuth)
      .values({
        id: ROW_IDS.rootKeySpace,
        workspaceId: ROW_IDS.rootWorkspace,
      })
      .onDuplicateKeyUpdate({ set: { createdAt: new Date() } });
    s.message("Created root keyspace");

    /**
     * Set up an api for production
     */
    await db
      .insert(schema.apis)
      .values({
        id: ROW_IDS.rootApi,
        name: "Unkey",
        workspaceId: ROW_IDS.rootWorkspace,
        authType: "key",
        keyAuthId: ROW_IDS.rootKeySpace,
        createdAt: new Date(),
      })
      .onDuplicateKeyUpdate({ set: { createdAt: new Date() } });
    s.message("Created root api");

    s.stop("seed done");
    return {
      workspace: {
        id: ROW_IDS.rootWorkspace,
      },
      api: { id: ROW_IDS.rootApi },
      webhooksApi: { id: ROW_IDS.webhookApi },
    };
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
        return mysqlDrizzle(conn, { schema, mode: "default" });
      } catch (e) {
        err = e as Error;
        await new Promise((r) => setTimeout(r, 1000 * i));
      }
    }

    throw err;
  });
}
