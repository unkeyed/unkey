import { pathToFileURL } from "node:url";
import {
  and,
  createCommentedPool,
  drizzle,
  eq,
  isNull,
  or,
  schema,
  staticTagsFromEnv,
} from "@unkey/db";
import { dns1035 } from "@unkey/id";

const READ_BATCH_SIZE = 1_000;

/** Matches both legacy representations of an unassigned Kubernetes namespace. */
export const missingK8sNamespace = or(
  isNull(schema.workspaces.k8sNamespace),
  eq(schema.workspaces.k8sNamespace, ""),
);

/**
 * Backfills Kubernetes namespaces for workspaces created before the field was required.
 * The migration updates NULL and empty values in batches and is safe to run more than once.
 *
 * Run from the repository root with `DRIZZLE_DATABASE_URL` set:
 * `mise exec -- pnpm --dir=web/tools/migrate workspace-k8s-namespace`
 */
async function main(): Promise<void> {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool(
    { uri: databaseUrl },
    staticTagsFromEnv("workspace-k8s-namespace-migration"),
  );
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    let updated = 0;

    while (true) {
      const rows = await db
        .select({ pk: schema.workspaces.pk })
        .from(schema.workspaces)
        .where(missingK8sNamespace)
        .orderBy(schema.workspaces.pk)
        .limit(READ_BATCH_SIZE);

      if (rows.length === 0) {
        break;
      }

      for (const row of rows) {
        const result = await db
          .update(schema.workspaces)
          .set({ k8sNamespace: dns1035() })
          .where(and(eq(schema.workspaces.pk, row.pk), missingK8sNamespace));
        updated += result[0].affectedRows;
      }

      console.info("progress", { updated });
    }

    console.info("Workspace Kubernetes namespace migration finished", { updated });
  } finally {
    await pool.end();
  }
}

const scriptPath = process.argv[1];
if (scriptPath && import.meta.url === pathToFileURL(scriptPath).href) {
  main().catch((error: unknown) => {
    console.error("Workspace Kubernetes namespace migration failed", error);
    process.exitCode = 1;
  });
}
