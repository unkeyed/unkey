"use client";
import {
  type QueryCollectionUtils,
  parseLoadSubsetOptions,
  queryCollectionOptions,
} from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";

import { plural } from "@/lib/fmt";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import type { EnvironmentVariable } from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { cachedAppEnvironmentIds } from "./app-environments";
import { trackSave } from "./pending-redeploy";
import { extractStringFilter } from "./utils";

const schema = z.object({
  // The API identifies a variable by key and returns no row id, so this key is
  // synthetic. The UI uses it only for React keys and row selection.
  id: z.string(),
  key: z.string(),
  // Empty for a writeonly variable. The API never returns its value.
  value: z.string(),
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable(),
  // A write replaces the row, so this shows the time of the last write.
  createdAt: z.number(),
  environmentId: z.string(),
  projectId: z.string(),
  appId: z.string(),
});

export type EnvVar = z.infer<typeof schema>;

/**
 * Environment variables collection.
 *
 * IMPORTANT: All queries MUST filter by projectId and appId:
 * .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId)))
 */
export const envVars = createCollection<EnvVar, string, QueryCollectionUtils<EnvVar, string>>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const appId = extractStringFilter(filters, "appId");
      return appId ? ["envVars", appId] : ["envVars"];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const appId = extractStringFilter(filters, "appId");
      const projectId = extractStringFilter(filters, "projectId");

      if (!appId || !projectId) {
        throw new Error(
          "Query must include eq(collection.projectId, projectId) and eq(collection.appId, appId) constraints",
        );
      }

      // The API lists the variables of one environment, so get the app's
      // environments first.
      const environmentIds = await cachedAppEnvironmentIds(projectId, appId);

      const perEnvironment = await Promise.all(
        environmentIds.map(async (environmentId) => {
          const variables = await listAllVariables(projectId, appId, environmentId);
          return variables.map((v) => toEnvVar(projectId, appId, environmentId, v));
        }),
      );

      return perEnvironment.flat();
    },
    getKey: (item) => item.id,
    id: "envVars",
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      const mutation = updateVariable(original, modified).catch(async (err) => {
        await envVars.utils.refetch().catch(() => {});
        throw err;
      });

      toast.promise(mutation, {
        loading: "Updating environment variable...",
        success: "Environment variable updated",
        error: (err) =>
          err instanceof RenameAppliedError
            ? envVarErrorToast(err.cause, `Renamed to ${err.newKey}, but the value change failed`)
            : envVarErrorToast(err, "Failed to update environment variable"),
      });

      await trackSave(mutation);
    },
    onDelete: async ({ transaction }) => {
      const originals = transaction.mutations.map((m) => m.original);
      const count = originals.length;
      const noun = count === 1 ? "Environment variable" : plural(count, "environment variable");

      const mutation = removeVariables(originals).catch(async (err) => {
        await envVars.utils.refetch().catch(() => {});
        throw err;
      });

      toast.promise(mutation, {
        loading: `Deleting ${noun.toLowerCase()}...`,
        success: `${noun} deleted`,
        error: (err) =>
          envVarErrorToast(err, `Failed to delete ${plural(count, "environment variable")}`),
      });

      await trackSave(mutation);
    },
  }),
);

/** Makes the collection key. The API returns no row id to use instead. */
export function envVarKey(environmentId: string, key: string): string {
  return `${environmentId}:${key}`;
}

function toEnvVar(
  projectId: string,
  appId: string,
  environmentId: string,
  v: EnvironmentVariable,
): EnvVar {
  return {
    id: envVarKey(environmentId, v.key),
    key: v.key,
    value: v.value ?? "",
    type: v.kind,
    description: v.description ?? null,
    createdAt: v.createdAt,
    environmentId,
    projectId,
    appId,
  };
}

async function listAllVariables(
  projectId: string,
  appId: string,
  environmentId: string,
): Promise<EnvironmentVariable[]> {
  const all: EnvironmentVariable[] = [];
  const pages = await getUnkeyClient().environments.listEnvironmentVariables({
    project: projectId,
    app: appId,
    environment: environmentId,
  });
  for await (const { result } of pages) {
    all.push(...result.data);
    if (!result.pagination.hasMore) {
      break;
    }
  }
  return all;
}

const MAX_VARIABLES_PER_REQUEST = 50;

async function inChunks<T>(items: T[], write: (chunk: T[]) => Promise<unknown>): Promise<void> {
  for (let i = 0; i < items.length; i += MAX_VARIABLES_PER_REQUEST) {
    await write(items.slice(i, i + MAX_VARIABLES_PER_REQUEST));
  }
}

export const MAX_VARIABLES = 200;

export type VariableInput = { key: string; value: string; kind: EnvVar["type"] };

export type AddEnvVarsInput = {
  projectId: string;
  appId: string;
  environmentIds: string[];
  variables: VariableInput[];
};

export type AddEnvVarsResult =
  | { status: "taken"; keys: string[] }
  | { status: "added"; count: number };

export async function addVariables({
  projectId,
  appId,
  environmentIds,
  variables,
}: AddEnvVarsInput): Promise<AddEnvVarsResult> {
  const submitted = new Set(variables.map((v) => v.key));
  const existing = await Promise.all(
    environmentIds.map((environmentId) => listAllVariables(projectId, appId, environmentId)),
  );
  const taken = [
    ...new Set(
      existing
        .flat()
        .map((v) => v.key)
        .filter((key) => submitted.has(key)),
    ),
  ].sort();
  if (taken.length > 0) {
    await envVars.utils.refetch().catch(() => {});
    return { status: "taken", keys: taken };
  }

  await writeEnvironments(environmentIds, async (environmentId, sent) => {
    await inChunks(variables, async (chunk) => {
      await getUnkeyClient().environments.setEnvironmentVariables({
        project: projectId,
        app: appId,
        environment: environmentId,
        variables: chunk,
      });
      sent();
    });
  });
  return { status: "added", count: variables.length };
}

export type SaveEnvVarsInput = AddEnvVarsInput & { edited: EnvVar[]; removed: EnvVar[] };

export async function saveVariables({
  edited,
  removed,
  ...add
}: SaveEnvVarsInput): Promise<AddEnvVarsResult> {
  const edits = groupByEnvironment(edited);
  const removals = groupByEnvironment(removed);
  const environmentIds = [...new Set([...removals.keys(), ...edits.keys()])];
  if (environmentIds.length > 0) {
    const client = getUnkeyClient().environments;
    await writeEnvironments(environmentIds, async (environment, sent) => {
      const target = { project: add.projectId, app: add.appId, environment };
      await inChunks(
        (removals.get(environment) ?? []).map((v) => v.key),
        async (chunk) => {
          await client.removeEnvironmentVariables({ ...target, variables: chunk });
          sent();
        },
      );
      await inChunks(
        (edits.get(environment) ?? []).map((v) => ({ key: v.key, value: v.value, kind: v.type })),
        async (chunk) => {
          await client.setEnvironmentVariables({ ...target, variables: chunk });
          sent();
        },
      );
    }).catch((error: unknown) => {
      throw error instanceof PartialAddError ? error.cause : error;
    });
  }
  return add.variables.length > 0 ? addVariables(add) : { status: "added", count: 0 };
}

async function writeEnvironments(
  environmentIds: string[],
  write: (environmentId: string, sent: () => void) => Promise<void>,
): Promise<void> {
  let wroteAny = false;
  const writes = await Promise.allSettled(
    environmentIds.map((environmentId) =>
      write(environmentId, () => {
        wroteAny = true;
      }),
    ),
  );
  const failed = writes.find((w): w is PromiseRejectedResult => w.status === "rejected");
  if (failed && !wroteAny) {
    await envVars.utils.refetch().catch(() => {});
    throw failed.reason;
  }
  await trackSave(envVars.utils.refetch().catch(() => {}));
  if (failed) {
    const savedEnvironmentIds = environmentIds.filter((_, i) => writes[i].status === "fulfilled");
    throw new PartialAddError(savedEnvironmentIds, failed.reason);
  }
}

export function envVarErrorToast(
  err: unknown,
  message: string,
): { message: string; description: string } {
  return getErrorToast(err, message, err instanceof Error ? err.message : undefined);
}

export async function makeVariablesSensitive(variables: EnvVar[]): Promise<number> {
  const [first] = variables;
  if (!first) {
    return 0;
  }
  try {
    const { updated } = await trpcClient.deploy.envVar.makeSensitive.mutate({
      appId: first.appId,
      targets: variables.map((v) => ({ environmentId: v.environmentId, key: v.key })),
    });
    return updated;
  } finally {
    await envVars.utils.refetch().catch(() => {});
  }
}

function groupByEnvironment(variables: readonly EnvVar[]): Map<string, EnvVar[]> {
  const byEnvironment = new Map<string, EnvVar[]>();
  for (const v of variables) {
    byEnvironment.set(v.environmentId, [...(byEnvironment.get(v.environmentId) ?? []), v]);
  }
  return byEnvironment;
}

async function removeVariables(variables: EnvVar[]): Promise<void> {
  await Promise.all(
    Array.from(groupByEnvironment(variables).values(), (group) =>
      inChunks(
        group.map((v) => v.key),
        (chunk) =>
          getUnkeyClient().environments.removeEnvironmentVariables({
            project: group[0].projectId,
            app: group[0].appId,
            environment: group[0].environmentId,
            variables: chunk,
          }),
      ),
    ),
  );
}

export class PartialAddError extends Error {
  constructor(
    readonly savedEnvironmentIds: string[],
    override readonly cause: unknown,
  ) {
    super(cause instanceof Error ? cause.message : "Some variables were not saved");
  }
}

export class RenameAppliedError extends Error {
  constructor(
    readonly newKey: string,
    override readonly cause: unknown,
  ) {
    super(cause instanceof Error ? cause.message : "The value change failed");
  }
}

export const WRITEONLY_VALUE_REQUIRED =
  "Type the new value to save a change to a sensitive variable.";

export async function updateVariable(original: EnvVar, modified: EnvVar): Promise<void> {
  const valueChanged = modified.value !== original.value;
  const otherFieldsChanged =
    modified.type !== original.type || modified.description !== original.description;
  if (original.type === "writeonly" && otherFieldsChanged && !valueChanged) {
    throw new Error(WRITEONLY_VALUE_REQUIRED);
  }

  if (modified.key !== original.key) {
    await trpcClient.deploy.envVar.rename.mutate({
      appId: modified.appId,
      environmentIds: [modified.environmentId],
      key: original.key,
      newKey: modified.key,
    });
  }
  if (!valueChanged && !otherFieldsChanged) {
    return;
  }

  try {
    await getUnkeyClient().environments.setEnvironmentVariables({
      project: modified.projectId,
      app: modified.appId,
      environment: modified.environmentId,
      variables: [
        {
          key: modified.key,
          value: modified.value,
          kind: modified.type,
          description: modified.description ?? undefined,
        },
      ],
    });
  } catch (err) {
    if (modified.key === original.key) {
      throw err;
    }
    throw new RenameAppliedError(modified.key, err);
  }
}
