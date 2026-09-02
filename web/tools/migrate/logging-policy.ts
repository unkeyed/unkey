import {
  createCommentedPool,
  type Database,
  drizzle,
  eq,
  gt,
  schema,
  staticTagsFromEnv,
} from "@unkey/db";
import { newUid } from "@unkey/id";

/**
 * Backfills a full-capture logging policy for apps that existed before header
 * and body capture became opt-in.
 *
 * The gateway always logs method, host, path, status, and latency. Header and
 * body capture now requires an enabled logging policy. Before this change the
 * gateway captured everything, so every existing sentinel config gets a
 * "Log everything" policy with all captures enabled and no match
 * conditions (no match conditions means all requests). This keeps the observed
 * behavior of existing apps unchanged.
 *
 * Two tables carry sentinel configs:
 * 1. `app_runtime_settings` — the live config the dashboard edits and new
 *    deployments snapshot.
 * 2. `deployments` — immutable snapshots the gateway serves from. Running
 *    deployments and rollback targets never re-read the live config, so the
 *    snapshots must be patched too.
 *
 * Safe to rerun: rows that already contain a logging policy are skipped.
 *
 * Run from the repository root with `DRIZZLE_DATABASE_URL` set:
 * `mise exec -- pnpm --dir=web/tools/migrate logging-policy`
 */
const READ_BATCH_SIZE = 1_000;

// Must stay <= POLICY_LIMITS.maxPolicies in the dashboard schema, or the
// dashboard can no longer read back and edit the patched config.
const MAX_POLICIES = 50;

/**
 * Returns the patched blob, or null when the row must not change (already has
 * a logging policy, config is unreadable, or the policy list is full).
 */
export function addLoggingPolicy(blob: string | null, rowRef: string): string | null {
  // The column is a longblob; mysql2 may hand back a Buffer at runtime, so
  // normalize through toString like the dashboard's loadPolicies does.
  const text = blob?.length ? blob.toString() : "{}";

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    console.error("skipping corrupt sentinel config", { rowRef });
    return null;
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    console.error("skipping non-object sentinel config", { rowRef });
    return null;
  }

  const config = parsed as { policies?: unknown };
  const policies = Array.isArray(config.policies) ? config.policies : [];

  const hasLoggingPolicy = policies.some(
    (p: unknown) => typeof p === "object" && p !== null && "logging" in p,
  );
  if (hasLoggingPolicy) {
    return null;
  }
  if (policies.length >= MAX_POLICIES) {
    console.error("skipping full policy list", { rowRef, count: policies.length });
    return null;
  }

  policies.push({
    id: newUid("policy"),
    name: "Log everything",
    enabled: true,
    logging: {
      requestHeaders: true,
      responseHeaders: true,
      requestBody: true,
      responseBody: true,
      query: true,
    },
  });

  return JSON.stringify({ ...config, policies });
}

type SentinelTable = typeof schema.appRuntimeSettings | typeof schema.deployments;

async function patchTable(db: Database, table: SentinelTable, tableName: string): Promise<void> {
  let cursor = 0;
  let scanned = 0;
  let patched = 0;

  while (true) {
    const rows = await db
      .select({ pk: table.pk, sentinelConfig: table.sentinelConfig })
      .from(table)
      .where(gt(table.pk, cursor))
      .orderBy(table.pk)
      .limit(READ_BATCH_SIZE);

    if (rows.length === 0) {
      break;
    }
    cursor = rows[rows.length - 1].pk;
    scanned += rows.length;

    for (const row of rows) {
      const blob = addLoggingPolicy(row.sentinelConfig, `${tableName}#${row.pk}`);
      if (blob === null) {
        continue;
      }
      await db
        .update(table)
        .set({ sentinelConfig: blob, updatedAt: Date.now() })
        .where(eq(table.pk, row.pk));
      patched++;
    }

    console.info("progress", { table: tableName, scanned, patched });
  }

  console.info("table done", { table: tableName, scanned, patched });
}

async function main() {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const pool = createCommentedPool(
    { uri: databaseUrl },
    staticTagsFromEnv("logging-policy-migration"),
  );
  const db = drizzle(pool, { schema, mode: "default" });

  try {
    await patchTable(db, schema.appRuntimeSettings, "app_runtime_settings");
    await patchTable(db, schema.deployments, "deployments");
  } finally {
    await pool.end();
  }
}

main().catch((error: unknown) => {
  console.error("Logging policy migration failed", error);
  process.exitCode = 1;
});
