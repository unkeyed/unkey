import type { UnkeyAuditLog } from "@/lib/audit";
import { type Transaction, and, eq, inArray, isDuplicateKeyError, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { match } from "@unkey/match";

export function takenKeys(existing: { key: string }[]): string[] {
  return [...new Set(existing.map((row) => row.key))].sort();
}

export async function lockAndFindTakenKeys(
  tx: Transaction,
  { appId, environmentIds, keys }: { appId: string; environmentIds: string[]; keys: string[] },
): Promise<string[]> {
  await tx
    .select({ id: schema.environments.id })
    .from(schema.environments)
    .where(inArray(schema.environments.id, environmentIds))
    .for("update");

  const existing = await tx
    .select({ key: schema.appEnvironmentVariables.key })
    .from(schema.appEnvironmentVariables)
    .where(
      and(
        eq(schema.appEnvironmentVariables.appId, appId),
        inArray(schema.appEnvironmentVariables.environmentId, environmentIds),
        inArray(schema.appEnvironmentVariables.key, keys),
      ),
    );
  return takenKeys(existing);
}

export function keyTakenError(key?: string): TRPCError {
  return new TRPCError({
    code: "CONFLICT",
    message: `A variable ${key ? `named "${key}"` : "with this name"} already exists in one of these environments`,
  });
}

// The v2 API does not take the environment lock, so the unique index is the
// real guard against a duplicate key.
export function toEnvVarWriteError(error: unknown, message: string): TRPCError {
  if (error instanceof TRPCError) {
    return error;
  }
  if (isDuplicateKeyError(error)) {
    return keyTakenError();
  }
  console.error(message, error);
  return new TRPCError({ code: "INTERNAL_SERVER_ERROR", message });
}

type EnvVarChange =
  | { action: "set"; key: string }
  | { action: "rename"; key: string; previousKey: string }
  | { action: "makeSensitive"; key: string };

function describeChange(change: EnvVarChange): string {
  return match(change)
    .with({ action: "set" }, ({ key }) => `Set environment variable ${key}`)
    .with(
      { action: "rename" },
      ({ key, previousKey }) => `Renamed environment variable ${previousKey} to ${key}`,
    )
    .with({ action: "makeSensitive" }, ({ key }) => `Made environment variable ${key} sensitive`)
    .exhaustive();
}

export function envVarAuditLogs({
  workspaceId,
  userId,
  context,
  changes,
}: {
  workspaceId: string;
  userId: string;
  context: UnkeyAuditLog["context"];
  changes: (EnvVarChange & { environment: { id: string; slug: string } })[];
}): UnkeyAuditLog[] {
  return changes.map(({ environment, ...change }) => ({
    workspaceId,
    actor: { type: "user", id: userId },
    event: "environment.update",
    description: `${describeChange(change)} for environment ${environment.id}`,
    resources: [
      {
        type: "environment",
        id: environment.id,
        name: environment.slug,
        meta:
          change.action === "rename"
            ? { key: change.key, previousKey: change.previousKey }
            : { key: change.key },
      },
    ],
    context,
  }));
}
