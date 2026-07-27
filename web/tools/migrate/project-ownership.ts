import { and, count, createCommentedPool, drizzle, eq, schema, staticTagsFromEnv } from "@unkey/db";
import { newId } from "@unkey/id";

/**
 * Creates missing exact-lowercase `default` projects and assigns them to owned
 * resources whose `project_id` is empty. Workspaces and updates are paginated
 * below Vitess limits, processed sequentially, and safe to rerun.
 *
 * Rollout sequence:
 * 1. Deploy the `project_id` columns and indexes.
 * 2. Run this migration once to create defaults and backfill existing rows.
 * 3. Deploy project-owned writers and wait for all previous instances to stop.
 * 4. Run this migration again to backfill rows created during the rollout.
 * 5. Verify all owned tables have no empty project IDs before removing it.
 *
 * Run from the repository root with `DRIZZLE_DATABASE_URL` set:
 * `mise exec -- pnpm --dir=web/tools/migrate project-ownership`
 */
const WORKSPACE_PAGE_SIZE = 1_000;
const UPDATE_BATCH_SIZE = 10_000;

async function main() {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool(
    { uri: databaseUrl, connectionLimit: 1 },
    staticTagsFromEnv("project-ownership-migration"),
  );
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    const startedAt = Date.now();
    const totals = await db.select({ workspaces: count() }).from(schema.workspaces);
    const totalWorkspaces = totals[0]?.workspaces ?? 0;
    let failures = 0;
    let processed = 0;
    let cursor = 0;

    console.info(`Starting project ownership migration for ${totalWorkspaces} workspaces`);

    while (true) {
      const workspaces = await db.query.workspaces.findMany({
        columns: { id: true, pk: true },
        where: (table, { gt }) => gt(table.pk, cursor),
        orderBy: (table, { asc }) => asc(table.pk),
        limit: WORKSPACE_PAGE_SIZE,
      });

      if (workspaces.length === 0) {
        break;
      }

      console.info("Processing workspace page", {
        cursor,
        pageSize: workspaces.length,
        processed,
        totalWorkspaces,
        failures,
      });

      for (const [workspaceIndex, workspace] of workspaces.entries()) {
        const currentWorkspace = processed + workspaceIndex + 1;
        console.info(
          `Migrating workspace ${workspace.id} (${currentWorkspace}/${totalWorkspaces}, ${((currentWorkspace / totalWorkspaces) * 100).toFixed(1)}%)`,
        );
        try {
          const project = await db.query.projects.findFirst({
            columns: { id: true, slug: true },
            where: (table, { eq }) =>
              and(eq(table.workspaceId, workspace.id), eq(table.slug, "default")),
          });

          let projectId = project?.slug === "default" ? project.id : undefined;
          if (!projectId) {
            projectId = newId("project");
            await db.insert(schema.projects).values({
              id: projectId,
              workspaceId: workspace.id,
              name: "Default",
              slug: "default",
              deleteProtection: true,
              createdAt: Date.now(),
            });
            console.info(`Created default project ${projectId} for workspace ${workspace.id}`);
          }

          let apis = 0;
          while (true) {
            const result = await db
              .update(schema.apis)
              .set({ projectId })
              .where(and(eq(schema.apis.workspaceId, workspace.id), eq(schema.apis.projectId, "")))
              .limit(UPDATE_BATCH_SIZE);
            apis += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          let keyAuth = 0;
          while (true) {
            const result = await db
              .update(schema.keyAuth)
              .set({ projectId })
              .where(
                and(eq(schema.keyAuth.workspaceId, workspace.id), eq(schema.keyAuth.projectId, "")),
              )
              .limit(UPDATE_BATCH_SIZE);
            keyAuth += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          let identities = 0;
          while (true) {
            const result = await db
              .update(schema.identities)
              .set({ projectId })
              .where(
                and(
                  eq(schema.identities.workspaceId, workspace.id),
                  eq(schema.identities.projectId, ""),
                ),
              )
              .limit(UPDATE_BATCH_SIZE);
            identities += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          let roles = 0;
          while (true) {
            const result = await db
              .update(schema.roles)
              .set({ projectId })
              .where(
                and(eq(schema.roles.workspaceId, workspace.id), eq(schema.roles.projectId, "")),
              )
              .limit(UPDATE_BATCH_SIZE);
            roles += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          let permissions = 0;
          while (true) {
            const result = await db
              .update(schema.permissions)
              .set({ projectId })
              .where(
                and(
                  eq(schema.permissions.workspaceId, workspace.id),
                  eq(schema.permissions.projectId, ""),
                ),
              )
              .limit(UPDATE_BATCH_SIZE);
            permissions += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          let ratelimitNamespaces = 0;
          while (true) {
            const result = await db
              .update(schema.ratelimitNamespaces)
              .set({ projectId })
              .where(
                and(
                  eq(schema.ratelimitNamespaces.workspaceId, workspace.id),
                  eq(schema.ratelimitNamespaces.projectId, ""),
                ),
              )
              .limit(UPDATE_BATCH_SIZE);
            ratelimitNamespaces += result[0].affectedRows;
            if (result[0].affectedRows < UPDATE_BATCH_SIZE) {
              break;
            }
          }

          console.info(`Backfilled workspace ${workspace.id}`, {
            apis,
            keyAuth,
            identities,
            roles,
            permissions,
            ratelimitNamespaces,
          });
        } catch (error) {
          failures++;
          console.error(`Failed to migrate workspace ${workspace.id}`, error);
        }
      }

      processed += workspaces.length;
      cursor = workspaces[workspaces.length - 1].pk;
      console.info("Completed workspace page", {
        cursor,
        processed,
        totalWorkspaces,
        progressPercent: Number(((processed / totalWorkspaces) * 100).toFixed(1)),
        failures,
      });

      if (workspaces.length < WORKSPACE_PAGE_SIZE) {
        break;
      }
    }

    console.info("Project ownership migration finished", {
      processed,
      totalWorkspaces,
      failures,
      elapsedSeconds: Number(((Date.now() - startedAt) / 1_000).toFixed(1)),
    });

    if (failures > 0) {
      throw new Error(`Failed to migrate ${failures} workspaces`);
    }
  } finally {
    await pool.end();
  }
}

main().catch((error: unknown) => {
  console.error("Project ownership migration failed", error);
  process.exitCode = 1;
});
