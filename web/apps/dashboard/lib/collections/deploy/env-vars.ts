"use client";
import {
  type QueryCollectionUtils,
  parseLoadSubsetOptions,
  queryCollectionOptions,
} from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";

import { plural } from "@/lib/fmt";
import type {
  AddEnvVarsInput,
  AddEnvVarsResult,
} from "@/lib/trpc/routers/deploy/env-vars/add-plan";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import type { EnvironmentVariable } from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { listAppEnvironments } from "./app-environments";
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
      const appEnvironments = await listAppEnvironments(projectId, appId);

      const perEnvironment = await Promise.all(
        appEnvironments.map(async (environment) => {
          const variables = await listAllVariables(projectId, appId, environment.id);
          return variables.map((v) => toEnvVar(projectId, appId, environment.id, v));
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
  let cursor: string | undefined;

  do {
    const page = await getUnkeyClient().environments.listEnvironmentVariables({
      project: projectId,
      app: appId,
      environment: environmentId,
      cursor,
    });
    all.push(...page.data);
    cursor = page.pagination?.hasMore ? page.pagination.cursor : undefined;
  } while (cursor);

  return all;
}

export type VariableInput = AddEnvVarsInput["variables"][number];

export async function addVariables(input: AddEnvVarsInput): Promise<AddEnvVarsResult> {
  const result = await trpcClient.deploy.envVar.add.mutate(input);
  if (result.status === "added") {
    await trackSave(envVars.utils.refetch().catch(() => undefined));
  }
  return result;
}

export function envVarErrorToast(
  err: unknown,
  message: string,
): { message: string; description: string } {
  return getErrorToast(err, message, err instanceof Error ? err.message : undefined);
}

/**
 * Turns recoverable variables writeonly, one way. The v2 API needs the value to
 * change the kind, so this goes through tRPC, which keeps the ciphertext.
 */
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

// The API rejects a request with more variables than this.
const MAX_VARIABLES_PER_REQUEST = 50;

/** Removes variables by key. Sends one request for each environment. */
async function removeVariables(variables: EnvVar[]): Promise<void> {
  const byEnvironment = new Map<string, EnvVar[]>();
  for (const v of variables) {
    const existing = byEnvironment.get(v.environmentId);
    if (existing) {
      existing.push(v);
    } else {
      byEnvironment.set(v.environmentId, [v]);
    }
  }

  await Promise.all(
    Array.from(byEnvironment.values(), async (group) => {
      const keys = group.map((v) => v.key);
      for (let i = 0; i < keys.length; i += MAX_VARIABLES_PER_REQUEST) {
        await getUnkeyClient().environments.removeEnvironmentVariables({
          project: group[0].projectId,
          app: group[0].appId,
          environment: group[0].environmentId,
          variables: keys.slice(i, i + MAX_VARIABLES_PER_REQUEST),
        });
      }
    }),
  );
}

/** The rename landed and the write after it failed. */
export class RenameAppliedError extends Error {
  constructor(
    readonly newKey: string,
    override readonly cause: unknown,
  ) {
    super(cause instanceof Error ? cause.message : "The value change failed");
  }
}

/**
 * Renames through tRPC, which changes only the key column, so the rename is
 * atomic and works for writeonly variables. Then writes the value, kind and
 * note if they changed.
 */
export async function updateVariable(original: EnvVar, modified: EnvVar): Promise<void> {
  if (modified.key === original.key) {
    return writeVariable(original, modified);
  }
  await trpcClient.deploy.envVar.rename.mutate({
    appId: modified.appId,
    environmentIds: [modified.environmentId],
    key: original.key,
    newKey: modified.key,
  });
  try {
    await writeVariable(original, modified);
  } catch (err) {
    throw new RenameAppliedError(modified.key, err);
  }
}

async function writeVariable(original: EnvVar, modified: EnvVar): Promise<void> {
  const valueChanged = modified.value !== original.value;
  if (
    !valueChanged &&
    modified.type === original.type &&
    modified.description === original.description
  ) {
    return;
  }

  // A write replaces the whole variable, so an untouched value is read again
  // instead of writing back the one loaded with the page.
  let value = modified.value;
  if (!valueChanged) {
    const stored = await listAllVariables(
      modified.projectId,
      modified.appId,
      modified.environmentId,
    );
    value = stored.find((v) => v.key === modified.key)?.value ?? value;
  }

  // The API merges nothing, so the kind and the note are sent every time.
  await getUnkeyClient().environments.setEnvironmentVariables({
    project: modified.projectId,
    app: modified.appId,
    environment: modified.environmentId,
    variables: [
      {
        key: modified.key,
        value,
        kind: modified.type,
        description: modified.description ?? undefined,
      },
    ],
  });
}
