"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const schema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(),
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable(),
  environmentId: z.string(),
  projectId: z.string(),
});

export type EnvVar = z.infer<typeof schema>;

/**
 * Environment variables collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ v }) => eq(v.projectId, projectId))
 */
export const envVars = createCollection<EnvVar, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const projectId = parseProjectIdFromWhere(opts.where);
      return projectId ? ["envVars", projectId] : ["envVars"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const projectId = parseProjectIdFromWhere(options?.where);

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      const data = await trpcClient.deploy.envVar.list.query({ projectId });

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
            environmentId,
            projectId,
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
          key: z.string().min(1),
          value: z.string().min(1),
          type: z.enum(["recoverable", "writeonly"]),
        })
        .parse(changes);

      const mutation = trpcClient.deploy.envVar.create.mutate({
        environmentId: insertInput.environmentId,
        variables: [
          {
            key: insertInput.key,
            value: insertInput.value,
            type: insertInput.type,
          },
        ],
      });

      toast.promise(mutation, {
        loading: "Creating environment variable...",
        success: "Environment variable created",
        error: (err) => ({
          message: "Failed to create environment variable",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      });

      await mutation;
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];

      const mutation = trpcClient.deploy.envVar.update.mutate({
        envVarId: original.id,
        key: modified.key,
        value: modified.value,
        type: modified.type,
      });

      toast.promise(mutation, {
        loading: "Updating environment variable...",
        success: "Environment variable updated",
        error: (err) => ({
          message: "Failed to update environment variable",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      });

      await mutation;
    },
    onDelete: async ({ transaction }) => {
      const { original } = transaction.mutations[0];

      const mutation = trpcClient.deploy.envVar.delete.mutate({
        envVarId: original.id,
      });

      toast.promise(mutation, {
        loading: "Deleting environment variable...",
        success: "Environment variable deleted",
        error: (err) => ({
          message: "Failed to delete environment variable",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      });

      await mutation;
    },
  }),
);
