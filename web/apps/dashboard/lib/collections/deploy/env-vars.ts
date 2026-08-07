"use client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";

import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import type { EnvironmentVariable } from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { z } from "zod";
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

      const mutation = renameAwareSet(original, modified);

      toast.promise(mutation, {
        loading: "Updating environment variable...",
        success: "Environment variable updated",
        error: (err) => ({
          message: "Failed to update environment variable",
          // The fallback carries the message of the rename check below.
          description: getErrorMessage(err, err instanceof Error ? err.message : undefined),
        }),
      });

      await trackSave(mutation);
    },
    onDelete: async ({ transaction }) => {
      const originals = transaction.mutations.map((m) => m.original);
      const count = originals.length;

      const mutation = removeVariables(originals);

      toast.promise(mutation, {
        loading: `Deleting ${count === 1 ? "environment variable" : `${count} environment variables`}...`,
        success: `${count === 1 ? "Environment variable" : `${count} environment variables`} deleted`,
        error: (err) => ({
          message: `Failed to delete environment variable${count === 1 ? "" : "s"}`,
          description: getErrorMessage(err),
        }),
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
 * An edit changes the value and the kind with the key, so it goes to the API.
 * The API cannot refuse a write onto a key that is in use, so this rejects the
 * rename against the rows that are already loaded.
 */
async function renameAwareSet(original: EnvVar, modified: EnvVar): Promise<unknown> {
  if (modified.key !== original.key) {
    const occupant = envVars.get(envVarKey(modified.environmentId, modified.key));
    if (occupant !== undefined && occupant.id !== original.id) {
      throw new Error(`A variable named "${modified.key}" already exists in this environment.`);
    }
  }

  await setVariables(modified.projectId, modified.appId, modified.environmentId, [
    {
      key: modified.key,
      value: modified.value,
      kind: modified.type,
      description: modified.description ?? undefined,
    },
  ]);

  if (modified.key !== original.key || modified.environmentId !== original.environmentId) {
    await removeVariables([original]);
  }

  return undefined;
}
