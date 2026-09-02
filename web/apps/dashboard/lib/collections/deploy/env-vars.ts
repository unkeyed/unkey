"use client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type { EnvironmentVariable } from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { queryClient, trpcClient } from "../client";
import { trackSave } from "./environment-settings";
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
export const envVars = createCollection<EnvVar, string>(
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
      const environments = await trpcClient.deploy.environment.list.query({ projectId });
      const appEnvironments = environments.filter((e) => e.appId === appId);

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

      const mutation = renameAwareSet(original, modified).catch(async (err) => {
        await envVars.utils.refetch().catch(() => {});
        throw err;
      });

      toast.promise(mutation, {
        loading: "Updating environment variable...",
        success: "Environment variable updated",
        error: (err) =>
          getErrorToast(
            err,
            "Failed to update environment variable",
            // The fallback carries the message of the rename check below.
            err instanceof Error ? err.message : undefined,
          ),
      });

      await trackSave(mutation);
    },
    onDelete: async ({ transaction }) => {
      const originals = transaction.mutations.map((m) => m.original);
      const count = originals.length;

      const mutation = removeVariables(originals).catch(async (err) => {
        await envVars.utils.refetch().catch(() => {});
        throw err;
      });

      toast.promise(mutation, {
        loading: `Deleting ${count === 1 ? "environment variable" : `${count} environment variables`}...`,
        success: `${count === 1 ? "Environment variable" : `${count} environment variables`} deleted`,
        error: (err) =>
          getErrorToast(err, `Failed to delete environment variable${count === 1 ? "" : "s"}`),
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

/** Lists the keys already set in the given environments. */
export async function listExistingKeys(
  projectId: string,
  appId: string,
  environmentIds: string[],
): Promise<{ key: string; environmentId: string }[]> {
  const perEnvironment = await Promise.all(
    environmentIds.map(async (environmentId) => {
      const variables = await listAllVariables(projectId, appId, environmentId);
      return variables.map((v) => ({ key: v.key, environmentId }));
    }),
  );

  return perEnvironment.flat();
}

export type VariableInput = {
  key: string;
  value: string;
  kind: EnvVar["type"];
  description?: string;
};

// The API rejects a request with more variables than this. Larger writes are
// sent in parts, and each part commits on its own.
const MAX_VARIABLES_PER_REQUEST = 50;

/**
 * Upserts variables in one environment. The API writes each entry exactly as
 * sent and merges nothing, so send the kind and the description every time.
 */
export async function setVariables(
  projectId: string,
  appId: string,
  environmentId: string,
  variables: VariableInput[],
): Promise<void> {
  for (let i = 0; i < variables.length; i += MAX_VARIABLES_PER_REQUEST) {
    await getUnkeyClient().environments.setEnvironmentVariables({
      project: projectId,
      app: appId,
      environment: environmentId,
      variables: variables.slice(i, i + MAX_VARIABLES_PER_REQUEST),
    });
  }
}

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

/**
 * Writes the modified variable, then removes the old key after a rename. The
 * two steps are not atomic. If the second fails, both keys stay.
 *
 * A write replaces the whole variable and the API refuses neither a key that is
 * in use nor a value older than the stored one, so both cases need the current
 * rows of the target environment. Reading them costs a request, so this reads
 * only when the edit can actually hit one of the two.
 */
async function renameAwareSet(original: EnvVar, modified: EnvVar): Promise<unknown> {
  const moved = modified.key !== original.key || modified.environmentId !== original.environmentId;
  // An untouched field still gets written back, which would undo a change made
  // elsewhere since this page loaded.
  const keepsLoadedValue = original.type === "recoverable" && modified.value === original.value;

  let value = modified.value;
  if (moved || keepsLoadedValue) {
    const target = await listAllVariables(
      modified.projectId,
      modified.appId,
      modified.environmentId,
    );
    if (moved && target.some((v) => v.key === modified.key)) {
      throw new Error(`A variable named "${modified.key}" already exists in this environment.`);
    }
    if (keepsLoadedValue) {
      value = target.find((v) => v.key === original.key)?.value ?? value;
    }
  }

  await setVariables(modified.projectId, modified.appId, modified.environmentId, [
    {
      key: modified.key,
      value,
      kind: modified.type,
      description: modified.description ?? undefined,
    },
  ]);

  if (moved) {
    await removeVariables([original]);
  }

  return undefined;
}
