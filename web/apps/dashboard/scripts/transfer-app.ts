#!/usr/bin/env bun
/**
 * Admin: move an app and all its project-scoped rows into a different project.
 *
 * project_id is denormalized across many tables (deployments, environments,
 * instances, custom_domains, frontline_routes, ...), so a transfer rewrites
 * project_id on every one of them inside a single transaction.
 *
 * Tables NOT touched (they follow the app automatically):
 *   - deployment_topology keys off deployment_id only, no project_id column.
 *   - app_build_settings / app_environment_variables / app_runtime_settings /
 *     app_regional_settings / portal_configurations key off app_id only.
 *
 * Dry-run by default. Pass --execute to apply.
 *
 * Usage:
 *   DATABASE_HOST=... DATABASE_USERNAME=... DATABASE_PASSWORD=... \
 *     bun run scripts/transfer-app.ts \
 *       --app-id app_xxx \
 *       --to-project-id proj_yyy \
 *       [--execute]
 */

import { parseArgs } from "node:util";
import { and, drizzle, eq, inArray, ne, schema, sql } from "@unkey/db";
import mysql from "mysql2/promise";

// Tables that denormalize project_id and are filtered by app_id.
const tablesScopedByApp = [
  { name: "environments", table: schema.environments },
  { name: "deployments", table: schema.deployments },
  { name: "instances", table: schema.instances },
  { name: "deployment_steps", table: schema.deploymentSteps },
  { name: "custom_domains", table: schema.customDomains },
  { name: "frontline_routes", table: schema.frontlineRoutes },
  { name: "cilium_network_policies", table: schema.ciliumNetworkPolicies },
  { name: "github_repo_connections", table: schema.githubRepoConnections },
] as const;

// Non-terminal deployment states. Transferring while any deployment sits here
// would race the Restate deploy workflow, which reads project_id mid-flight.
const activeDeploymentStatuses = [
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "awaiting_approval",
] as const;

async function main() {
  const { values } = parseArgs({
    options: {
      "app-id": { type: "string" },
      "to-project-id": { type: "string" },
      execute: { type: "boolean", default: false },
    },
  });

  const appId = values["app-id"];
  const toProjectId = values["to-project-id"];
  if (!appId || !toProjectId) {
    throw new Error("--app-id and --to-project-id are required");
  }

  const { DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD } = process.env;
  if (!DATABASE_HOST || !DATABASE_USERNAME || !DATABASE_PASSWORD) {
    throw new Error("DATABASE_HOST, DATABASE_USERNAME and DATABASE_PASSWORD must be set");
  }
  const isLocal = DATABASE_HOST.includes("localhost") || DATABASE_HOST.includes("127.0.0.1");

  const pool = mysql.createPool({
    host: DATABASE_HOST.split(":")[0],
    port: DATABASE_HOST.includes(":") ? Number(DATABASE_HOST.split(":")[1]) : 3306,
    user: DATABASE_USERNAME,
    password: DATABASE_PASSWORD,
    database: "unkey",
    connectionLimit: 5,
    ...(isLocal ? {} : { ssl: { rejectUnauthorized: true } }),
  });
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    // Load the app.
    const [app] = await db
      .select({
        workspaceId: schema.apps.workspaceId,
        projectId: schema.apps.projectId,
        slug: schema.apps.slug,
      })
      .from(schema.apps)
      .where(eq(schema.apps.id, appId))
      .limit(1);
    if (!app) {
      throw new Error(`app "${appId}" not found`);
    }

    // Load the destination project.
    const [project] = await db
      .select({ workspaceId: schema.projects.workspaceId, slug: schema.projects.slug })
      .from(schema.projects)
      .where(eq(schema.projects.id, toProjectId))
      .limit(1);
    if (!project) {
      throw new Error(`destination project "${toProjectId}" not found`);
    }

    // Invariants.
    if (app.projectId === toProjectId) {
      throw new Error(`app "${appId}" is already in project "${toProjectId}"`);
    }
    if (app.workspaceId !== project.workspaceId) {
      throw new Error(
        `cross-workspace transfer is not allowed: app is in workspace "${app.workspaceId}", destination project is in workspace "${project.workspaceId}"`,
      );
    }

    // apps_project_slug_idx is UNIQUE(project_id, slug): a colliding slug in the
    // destination would make the apps UPDATE fail mid-transaction.
    const [collision] = await db
      .select({ id: schema.apps.id })
      .from(schema.apps)
      .where(
        and(
          eq(schema.apps.projectId, toProjectId),
          eq(schema.apps.slug, app.slug),
          ne(schema.apps.id, appId),
        ),
      )
      .limit(1);
    if (collision) {
      throw new Error(
        `destination project "${toProjectId}" already has an app with slug "${app.slug}" (${collision.id}); rename one first`,
      );
    }

    const [{ active }] = await db
      .select({ active: sql<number>`count(*)` })
      .from(schema.deployments)
      .where(
        and(
          eq(schema.deployments.appId, appId),
          inArray(schema.deployments.status, [...activeDeploymentStatuses]),
        ),
      );
    if (Number(active) > 0) {
      throw new Error(
        `app has ${active} deployment(s) in a non-terminal state; wait for them to settle before transferring`,
      );
    }

    // Preview.
    console.info(
      `Transfer app ${appId} (slug="${app.slug}") in workspace ${app.workspaceId}\n` +
        `  from project ${app.projectId}\n` +
        `  to   project ${toProjectId} (slug="${project.slug}")\n`,
    );
    console.info("Rows to re-point (project_id):");
    console.info(`  ${"apps".padEnd(26)} 1`);
    for (const { name, table } of tablesScopedByApp) {
      const [{ c }] = await db
        .select({ c: sql<number>`count(*)` })
        .from(table)
        .where(eq(table.appId, appId));
      console.info(`  ${name.padEnd(26)} ${c}`);
    }

    if (!values.execute) {
      console.info("\nDry run. Re-run with --execute to apply.");
      return;
    }

    await db.transaction(async (tx) => {
      // The unique id filter guarantees this touches exactly the one app row.
      await tx.update(schema.apps).set({ projectId: toProjectId }).where(eq(schema.apps.id, appId));

      for (const { table } of tablesScopedByApp) {
        await tx.update(table).set({ projectId: toProjectId }).where(eq(table.appId, appId));
      }
    });

    console.info(`\nTransferred app ${appId} to project ${toProjectId}.`);
    console.info(
      "Existing public URLs embed the old project slug and keep routing; new deployments will use the new project slug.",
    );
  } finally {
    await pool.end();
  }
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : err);
  process.exit(1);
});
