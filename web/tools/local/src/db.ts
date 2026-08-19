import { randomUUID } from "node:crypto";
import { and, drizzle, eq, schema, sql } from "@unkey/db";
import mysql from "mysql2/promise";
import { task } from "./util";

const ROW_IDS = {
  rootWorkspace: "ws_local_root",
  rootProject: "proj_local_root",
  rootKeySpace: "ks_local_root_keys",
  rootApi: "api_local_root_keys",
  webhookKeySpace: "ks_local_webhook_keys",
  webhookApi: "api_local_webhook_keys",
};

export async function prepareDatabase(): Promise<{
  workspace: { id: string };
  api: { id: string };
  webhooksApi: { id: string };
}> {
  const db = await connectDatabase();

  return await task("Seeding database", async (s) => {
    // root workspace
    await db
      .insert(schema.workspaces)
      .values({
        id: ROW_IDS.rootWorkspace,
        orgId: "user_REPLACE_ME",
        name: "Unkey",
        slug: `unkey-workspace-${randomUUID()}`,
        createdAtM: Date.now(),
        betaFeatures: {},
      })
      .onDuplicateKeyUpdate({ set: { createdAtM: Date.now() } });

    await db
      .insert(schema.limits)
      .values({
        workspaceId: ROW_IDS.rootWorkspace,
        apiBillableOperationsCountMaxPerMonth: 150_000,
        apiRequestsCountMaxPerMinute: null,
        logsRetentionDaysMax: 7,
        logsAuditRetentionDaysMax: 30,
        teamEnabled: false,
        cpuCoresMax: 10,
        cpuCoresMaxPerInstance: 2,
        memoryMibMax: 20_480,
        memoryMibMaxPerInstance: 4_096,
        storageMibMax: 51_200,
        storageMibMaxPerInstance: 10_240,
        buildsConcurrentMax: 1,
        customDomainsMax: 0,
        autoscalingReplicasMax: 0,
      })
      .onDuplicateKeyUpdate({ set: { workspaceId: ROW_IDS.rootWorkspace } });

    s.message("Created root workspace");

    await db
      .insert(schema.projects)
      .values({
        id: ROW_IDS.rootProject,
        workspaceId: ROW_IDS.rootWorkspace,
        name: "Default",
        slug: "default",
        deleteProtection: true,
      })
      .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } });
    s.message("Created root default project");

    const [defaultProject] = await db
      .select({ id: schema.projects.id })
      .from(schema.projects)
      .where(
        and(
          eq(schema.projects.workspaceId, ROW_IDS.rootWorkspace),
          sql`BINARY ${schema.projects.slug} = 'default'`,
        ),
      )
      .limit(1);
    if (!defaultProject) {
      throw new Error(`Default project not found for workspace ${ROW_IDS.rootWorkspace}`);
    }

    await db
      .insert(schema.keyAuth)
      .values({
        id: ROW_IDS.rootKeySpace,
        workspaceId: ROW_IDS.rootWorkspace,
        projectId: defaultProject.id,
      })
      .onDuplicateKeyUpdate({ set: { createdAtM: Date.now() } });
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
        projectId: defaultProject.id,
        authType: "key",
        keyAuthId: ROW_IDS.rootKeySpace,
        createdAtM: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { createdAtM: Date.now() } });
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
        return drizzle(conn, { schema, mode: "default" });
      } catch (e) {
        err = e as Error;
        await new Promise((r) => setTimeout(r, 1000 * i));
      }
    }

    throw err;
  });
}
