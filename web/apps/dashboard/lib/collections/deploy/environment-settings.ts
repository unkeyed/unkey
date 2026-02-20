"use client";
import type { SentinelConfig } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseEnvironmentIdFromWhere, validateEnvironmentIdInQuery } from "./utils";

const healthcheckSchema = z
  .object({
    method: z.enum(["GET", "POST"]),
    path: z.string(),
    intervalSeconds: z.number(),
    timeoutSeconds: z.number(),
    failureThreshold: z.number(),
    initialDelaySeconds: z.number(),
  })
  .nullable();

const sentinelConfigSchema: z.ZodType<SentinelConfig | undefined> = z
  .object({
    policies: z.array(
      z.object({
        id: z.string(),
        name: z.string(),
        enabled: z.boolean(),
        keyauth: z.object({ keySpaceIds: z.array(z.string()) }),
      }),
    ),
  })
  .optional();

const schema = z.object({
  environmentId: z.string(),
  // Build settings
  dockerfile: z.string(),
  dockerContext: z.string(),
  // Runtime settings
  port: z.number().int(),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  command: z.array(z.string()),
  healthcheck: healthcheckSchema,
  regionConfig: z.record(z.string(), z.number()),
  shutdownSignal: z.string(),
  sentinelConfig: sentinelConfigSchema,
});

export type EnvironmentSettings = z.infer<typeof schema>;

/**
 * Environment settings collection - flattened build + runtime settings.
 *
 * IMPORTANT: All queries MUST filter by environmentId:
 * .where(({ s }) => eq(s.environmentId, environmentId))
 */
export const environmentSettings = createCollection<EnvironmentSettings, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const environmentId = parseEnvironmentIdFromWhere(opts.where);
      return environmentId ? ["environmentSettings", environmentId] : ["environmentSettings"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateEnvironmentIdInQuery(options?.where);
      const environmentId = parseEnvironmentIdFromWhere(options?.where);

      if (!environmentId) {
        throw new Error(
          "Query must include eq(collection.environmentId, environmentId) constraint",
        );
      }

      const result = await trpcClient.deploy.environmentSettings.get.query({
        environmentId,
      });

      const build = result.buildSettings;
      const runtime = result.runtimeSettings;

      return [
        {
          environmentId,
          dockerfile: build?.dockerfile ?? "Dockerfile",
          dockerContext: build?.dockerContext ?? ".",
          port: runtime?.port ?? 8080,
          cpuMillicores: runtime?.cpuMillicores ?? 256,
          memoryMib: runtime?.memoryMib ?? 256,
          command: (runtime?.command as string[] | undefined) ?? [],
          healthcheck: runtime?.healthcheck ?? null,
          regionConfig: (runtime?.regionConfig as Record<string, number> | undefined) ?? {},
          shutdownSignal: "SIGTERM",
          sentinelConfig: runtime?.sentinelConfig,
        },
      ];
    },
    getKey: (item) => item.environmentId,
    id: "environmentSettings",
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];
      const environmentId = original.environmentId;

      const mutations: Promise<unknown>[] = [];

      if (modified.dockerfile !== original.dockerfile) {
        mutations.push(
          trpcClient.deploy.environmentSettings.build.updateDockerfile.mutate({
            environmentId,
            dockerfile: modified.dockerfile,
          }),
        );
      }

      if (modified.dockerContext !== original.dockerContext) {
        mutations.push(
          trpcClient.deploy.environmentSettings.build.updateDockerContext.mutate({
            environmentId,
            dockerContext: modified.dockerContext,
          }),
        );
      }

      if (modified.port !== original.port) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updatePort.mutate({
            environmentId,
            port: modified.port,
          }),
        );
      }

      if (modified.cpuMillicores !== original.cpuMillicores) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateCpu.mutate({
            environmentId,
            cpuMillicores: modified.cpuMillicores,
          }),
        );
      }

      if (modified.memoryMib !== original.memoryMib) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateMemory.mutate({
            environmentId,
            memoryMib: modified.memoryMib,
          }),
        );
      }

      if (JSON.stringify(modified.command) !== JSON.stringify(original.command)) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateCommand.mutate({
            environmentId,
            command: modified.command,
          }),
        );
      }

      if (JSON.stringify(modified.healthcheck) !== JSON.stringify(original.healthcheck)) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateHealthcheck.mutate({
            environmentId,
            healthcheck: modified.healthcheck,
          }),
        );
      }

      // Region keys changed → updateRegions
      const origRegions = Object.keys(original.regionConfig).sort();
      const modRegions = Object.keys(modified.regionConfig).sort();
      if (JSON.stringify(origRegions) !== JSON.stringify(modRegions)) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateRegions.mutate({
            environmentId,
            regions: modRegions,
          }),
        );
      }

      // Region values changed → updateInstances (all regions share same count)
      const origValues = Object.values(original.regionConfig);
      const modValues = Object.values(modified.regionConfig);
      if (
        origValues.length === modValues.length &&
        JSON.stringify(origRegions) === JSON.stringify(modRegions) &&
        modValues.length > 0 &&
        modValues[0] !== origValues[0]
      ) {
        mutations.push(
          trpcClient.deploy.environmentSettings.runtime.updateInstances.mutate({
            environmentId,
            replicasPerRegion: modValues[0],
          }),
        );
      }

      if (JSON.stringify(modified.sentinelConfig) !== JSON.stringify(original.sentinelConfig)) {
        const keyspaceIds: string[] = [];
        for (const policy of modified.sentinelConfig?.policies ?? []) {
          if (policy.keyauth) {
            keyspaceIds.push(...policy.keyauth.keySpaceIds);
          }
        }
        mutations.push(
          trpcClient.deploy.environmentSettings.sentinel.updateMiddleware.mutate({
            environmentId,
            keyspaceIds,
          }),
        );
      }

      if (mutations.length === 0) {
        return;
      }

      const allMutations = Promise.all(mutations);
      toast.promise(allMutations, {
        loading: "Saving settings...",
        success: "Settings updated",
        error: (err) => ({
          message: "Failed to update settings",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      });
      await allMutations;
    },
  }),
);
