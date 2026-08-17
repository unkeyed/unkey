import { pathToFileURL } from "node:url";
import {
  type Database,
  createCommentedPool,
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
 * Both environments for an app receive the same policy ID so the dashboard
 * presents them as one policy enabled in both environments.
 *
 * Safe to rerun: logging policies created by an earlier version of this
 * migration are reconciled to the same ID. Other logging policies are skipped.
 *
 * Run from the repository root with `DRIZZLE_DATABASE_URL` set:
 * `mise exec -- pnpm --dir=web/tools/migrate logging-policy`
 */
const READ_BATCH_SIZE = 1_000;

// Must stay <= SENTINEL_LIMITS.maxPolicies in the dashboard schema, or the
// dashboard can no longer read back and edit the patched config.
const MAX_POLICIES = 50;

type ReconcileResult = {
  blob: string | null;
  policyId: string | undefined;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isBackfilledLoggingPolicy(policy: unknown): policy is Record<string, unknown> & {
  id: string;
} {
  if (!isRecord(policy) || typeof policy.id !== "string") {
    return false;
  }
  if (policy.name !== "Log everything" || policy.enabled !== true || "match" in policy) {
    return false;
  }
  if (!isRecord(policy.logging) || Object.keys(policy.logging).length !== 5) {
    return false;
  }
  return (
    policy.logging.requestHeaders === true &&
    policy.logging.responseHeaders === true &&
    policy.logging.requestBody === true &&
    policy.logging.responseBody === true &&
    policy.logging.query === true
  );
}

/**
 * Adds the backfill policy or reconciles an earlier backfill to policyId.
 * Returns a null blob when the row must not change.
 */
export function reconcileLoggingPolicy(
  blob: string | null,
  rowRef: string,
  policyId?: string,
): ReconcileResult {
  // The column is a longblob; mysql2 may hand back a Buffer at runtime, so
  // normalize through toString like the dashboard's loadPolicies does.
  const text = blob?.length ? blob.toString() : "{}";

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    console.error("skipping corrupt sentinel config", { rowRef });
    return { blob: null, policyId };
  }
  if (!isRecord(parsed)) {
    console.error("skipping non-object sentinel config", { rowRef });
    return { blob: null, policyId };
  }

  const policies = Array.isArray(parsed.policies) ? parsed.policies : [];

  const backfilledPolicy = policies.find(isBackfilledLoggingPolicy);
  if (backfilledPolicy) {
    const canonicalPolicyId = policyId ?? backfilledPolicy.id;
    if (backfilledPolicy.id === canonicalPolicyId) {
      return { blob: null, policyId: canonicalPolicyId };
    }

    policies[policies.indexOf(backfilledPolicy)] = { ...backfilledPolicy, id: canonicalPolicyId };
    return {
      blob: JSON.stringify({ ...parsed, policies }),
      policyId: canonicalPolicyId,
    };
  }

  const hasLoggingPolicy = policies.some(
    (p: unknown) => typeof p === "object" && p !== null && "logging" in p,
  );
  if (hasLoggingPolicy) {
    return { blob: null, policyId };
  }
  if (policies.length >= MAX_POLICIES) {
    console.error("skipping full policy list", { rowRef, count: policies.length });
    return { blob: null, policyId };
  }

  const canonicalPolicyId = policyId ?? newUid("policy");
  policies.push({
    id: canonicalPolicyId,
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

  return {
    blob: JSON.stringify({ ...parsed, policies }),
    policyId: canonicalPolicyId,
  };
}

type SentinelTable = typeof schema.appRuntimeSettings | typeof schema.deployments;

async function patchTable(
  db: Database,
  table: SentinelTable,
  tableName: string,
  policyIdsByApp: Map<string, string>,
): Promise<void> {
  let cursor = 0;
  let scanned = 0;
  let patched = 0;

  while (true) {
    const rows = await db
      .select({ pk: table.pk, appId: table.appId, sentinelConfig: table.sentinelConfig })
      .from(table)
      .where(gt(table.pk, cursor))
      .orderBy(table.pk)
      .limit(READ_BATCH_SIZE);

    if (rows.length === 0) {
      break;
    }
    cursor = rows[rows.length - 1].pk;
    scanned += rows.length;

    const pendingUpdates: { pk: number; sentinelConfig: string }[] = [];
    for (const row of rows) {
      const result = reconcileLoggingPolicy(
        row.sentinelConfig,
        `${tableName}#${row.pk}`,
        policyIdsByApp.get(row.appId),
      );
      if (result.policyId) {
        policyIdsByApp.set(row.appId, result.policyId);
      }
      if (result.blob === null) {
        continue;
      }
      pendingUpdates.push({ pk: row.pk, sentinelConfig: result.blob });
    }

    // The pool bounds active connections, so enqueue the batch together rather
    // than waiting for every database round trip in sequence.
    await Promise.all(
      pendingUpdates.map((update) =>
        db
          .update(table)
          .set({ sentinelConfig: update.sentinelConfig, updatedAt: Date.now() })
          .where(eq(table.pk, update.pk)),
      ),
    );
    patched += pendingUpdates.length;

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
    const policyIdsByApp = new Map<string, string>();
    await patchTable(db, schema.appRuntimeSettings, "app_runtime_settings", policyIdsByApp);
    await patchTable(db, schema.deployments, "deployments", policyIdsByApp);
  } finally {
    await pool.end();
  }
}

const scriptPath = process.argv[1];
if (scriptPath && import.meta.url === pathToFileURL(scriptPath).href) {
  main().catch((error: unknown) => {
    console.error("Logging policy migration failed", error);
    process.exitCode = 1;
  });
}
