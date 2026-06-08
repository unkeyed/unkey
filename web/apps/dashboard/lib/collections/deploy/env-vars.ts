"use client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";

import { envVarKeySchema } from "@/lib/schemas/env-var";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { trackSave } from "./environment-settings";

type ParsedFilter = { field: Array<string | number>; operator: string; value?: unknown };

function extractStringFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "string" ? value : undefined;
}

const schema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(),
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable(),
  updatedAt: z.number(),
  environmentId: z.string(),
  appId: z.string(),
});

export type EnvVar = z.infer<typeof schema>;

/**
 * Environment variables collection.
 *
 * IMPORTANT: All queries MUST filter by appId:
 * .where(({ v }) => eq(v.appId, appId))
 */
export const envVars = createCollection<EnvVar, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const appId = extractStringFilter(filters, "appId", "eq");
      return appId ? ["envVars", appId] : ["envVars"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      const { filters } = parseLoadSubsetOptions(options);
      const appId = extractStringFilter(filters, "appId", "eq");

      if (!appId) {
        throw new Error("Query must include eq(collection.appId, appId) constraint");
      }

      const data = await trpcClient.deploy.envVar.list.query({ appId });

      const result: EnvVar[] = [];
      for (const [_slug, envData] of Object.entries(data)) {
        const environmentId = envData.id;
        for (const v of envData.variables) {
          result.push({
            id: v.id,
            key: v.key,
            value: v.value,
            type: v.type,
            description: v.description,
            updatedAt: v.updatedAt,
            environmentId,
            appId: v.appId,
          });
        }
      }

      return result;
    },
    getKey: (item) => item.id,
    id: "envVars",
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const insertInput = z
        .object({
          environmentId: z.string().min(1),
          key: envVarKeySchema,
          value: z.string().min(1),
          type: z.enum(["recoverable", "writeonly"]),
          description: z.string().nullable().optional(),
        })
        .parse(changes);

      const mutation = trpcClient.deploy.envVar.create.mutate({
        environmentId: insertInput.environmentId,
        variables: [
          {
            key: insertInput.key,
            value: insertInput.value,
            type: insertInput.type,
            description: insertInput.description ?? null,
          },
        ],
      });

      await trackSave(mutation);
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      const mutation = trpcClient.deploy.envVar.update.mutate({
        envVarId: original.id,
        environmentId: modified.environmentId,
        key: modified.key,
        value: modified.value,
        type: modified.type,
        description: modified.description,
      });

      toast.promise(mutation, {
        loading: "Updating environment variable...",
        success: "Environment variable updated",
        error: (err) => ({
          message: "Failed to update environment variable",
          description: err.message,
        }),
      });

      await trackSave(mutation);
    },
    onDelete: async ({ transaction }) => {
      const envVarIds = transaction.mutations.map((m) => m.original.id);
      const count = envVarIds.length;

      const mutation = trpcClient.deploy.envVar.delete.mutate({ envVarIds });

      toast.promise(mutation, {
        loading: `Deleting ${count === 1 ? "environment variable" : `${count} environment variables`}...`,
        success: `${count === 1 ? "Environment variable" : `${count} environment variables`} deleted`,
        error: (err) => ({
          message: `Failed to delete environment variable${count === 1 ? "" : "s"}`,
          description: err.message,
        }),
      });

      await trackSave(mutation);
    },
  }),
);
