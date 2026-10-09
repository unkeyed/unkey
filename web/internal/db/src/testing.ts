import { createRequire } from "node:module";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";
import { onTestFinished } from "vitest";
import { drizzle, schema } from "./index";

const { generateMySQLDrizzleJson, generateMySQLMigration }: typeof import("drizzle-kit/api") =
  createRequire(import.meta.url)("drizzle-kit/api");

export async function connectionTestDatabase(databaseUrl: string) {
  const url = new URL(databaseUrl);
  if (!["localhost", "127.0.0.1", "[::1]"].includes(url.hostname)) {
    throw new Error("Connection tests require a local disposable database");
  }
  const name = newId("test");
  const admin = mysql.createPool(databaseUrl);
  let pool: mysql.Pool | undefined;
  onTestFinished(async () => {
    try {
      await pool?.end();
    } finally {
      try {
        await admin.query(`DROP DATABASE IF EXISTS \`${name}\``);
      } finally {
        await admin.end();
      }
    }
  });
  await admin.query(`CREATE DATABASE \`${name}\``);
  url.pathname = `/${name}`;
  pool = mysql.createPool(url.toString());
  const database = drizzle(pool, { schema, mode: "default" });
  const tables = {
    apps: schema.apps,
    projects: schema.projects,
    environments: schema.environments,
    deployments: schema.deployments,
    appConnections: schema.appConnections,
    connectionAppTargets: schema.connectionAppTargets,
    deploymentConnections: schema.deploymentConnections,
    deploymentConnectionAppTargets: schema.deploymentConnectionAppTargets,
  };
  const empty = await generateMySQLDrizzleJson({});
  const desired = await generateMySQLDrizzleJson(tables);
  const statements = await generateMySQLMigration(empty, desired);
  if (statements.length === 0) {
    throw new Error("Drizzle generated no connection test tables");
  }
  for (const statement of statements) {
    await pool.query(statement);
  }
  return database;
}

export function deploymentSeed(
  row: Pick<
    typeof schema.deployments.$inferInsert,
    "id" | "workspaceId" | "projectId" | "appId" | "environmentId"
  > &
    Partial<typeof schema.deployments.$inferInsert>,
): typeof schema.deployments.$inferInsert {
  return {
    k8sName: row.id,
    sentinelConfig: "",
    encryptedEnvironmentVariables: "",
    cpuMillicores: 100,
    memoryMib: 128,
    ...row,
  };
}
