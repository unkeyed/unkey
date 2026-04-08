import {
  type SentinelPolicy,
  fromWirePolicy,
  sentinelConfigSchema,
  toWirePolicy,
} from "@/lib/collections/deploy/sentinel-policies.schema";
/**
 * Shared helpers for sentinel policy tRPC endpoints.
 *
 * The DB stores all sentinel policies for an environment as a single JSON blob
 * in `appRuntimeSettings.sentinelConfig`. Per-policy endpoints do read-modify-write
 * on that blob — every mutation re-validates the *entire* policies array before
 * persisting so a corrupted row can never enter the DB.
 */
import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";

/** A drizzle executor — either the pool `db` or a transactional `tx`. */
export type Executor = typeof db | Parameters<Parameters<typeof db.transaction>[0]>[0];

/**
 * Look up an environment scoped to the caller's workspace.
 * Throws NOT_FOUND if missing or not owned by this workspace.
 */
export async function loadOwnedEnvironment(
  workspaceId: string,
  environmentId: string,
  executor: Executor = db,
): Promise<{ appId: string }> {
  const env = await executor.query.environments.findFirst({
    where: and(eq(environments.id, environmentId), eq(environments.workspaceId, workspaceId)),
    columns: { appId: true },
  });
  if (!env) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
  }
  return env;
}

/**
 * Verify every keyspace ID belongs to the caller's workspace.
 * Throws NOT_FOUND on the first one that doesn't.
 */
export async function assertKeyspacesOwned(
  workspaceId: string,
  keyspaceIds: readonly string[],
): Promise<void> {
  if (keyspaceIds.length === 0) {
    return;
  }
  const rows = await db.query.keyAuth
    .findMany({
      where: (table, { and, inArray }) =>
        and(inArray(table.id, [...keyspaceIds]), eq(table.workspaceId, workspaceId)),
      columns: { id: true },
    })
    .catch((err) => {
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Unable to verify keyspaces",
      });
    });
  const owned = new Set(rows.map((r) => r.id));
  for (const id of keyspaceIds) {
    if (!owned.has(id)) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Keyspace ${id} does not exist`,
      });
    }
  }
}

/**
 * Read the current sentinelConfig blob for an environment, returning the
 * decoded list of policies. Returns [] when the row doesn't exist yet or the
 * blob is empty.
 */
export async function loadPolicies(
  workspaceId: string,
  environmentId: string,
  executor: Executor = db,
): Promise<SentinelPolicy[]> {
  const row = await executor.query.appRuntimeSettings.findFirst({
    where: and(
      eq(appRuntimeSettings.workspaceId, workspaceId),
      eq(appRuntimeSettings.environmentId, environmentId),
    ),
    columns: { sentinelConfig: true },
  });
  if (!row?.sentinelConfig?.length) {
    return [];
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(Buffer.from(row.sentinelConfig).toString());
  } catch (err) {
    console.error("corrupt sentinelConfig blob", err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Stored sentinel config is corrupted",
    });
  }
  if (typeof parsed !== "object" || parsed === null) {
    return [];
  }
  const maybePolicies = (parsed as { policies?: unknown }).policies;
  if (!Array.isArray(maybePolicies)) {
    return [];
  }
  // Re-attach client-side `type` discriminator and validate each policy.
  const policies: SentinelPolicy[] = [];
  for (const raw of maybePolicies) {
    policies.push(fromWirePolicy(raw));
  }
  return policies;
}

/**
 * Persist a new policies array for an environment. Re-validates the whole
 * array against `sentinelConfigSchema` before writing — corrupt blobs can't
 * land in the DB.
 */
export async function savePolicies(
  workspaceId: string,
  environmentId: string,
  appId: string,
  policies: SentinelPolicy[],
  executor: Executor = db,
): Promise<void> {
  sentinelConfigSchema.parse({ policies });

  const blob = JSON.stringify({ policies: policies.map(toWirePolicy) });
  const now = Date.now();

  await executor
    .insert(appRuntimeSettings)
    .values({
      workspaceId,
      appId,
      environmentId,
      sentinelConfig: blob,
      createdAt: now,
      updatedAt: now,
    })
    .onDuplicateKeyUpdate({ set: { sentinelConfig: blob, updatedAt: now } });
}
