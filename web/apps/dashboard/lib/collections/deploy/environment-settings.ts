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
    // Setting don't change that often and we already do revalidation on submit so no need to poll it short
    refetchInterval: 30_000,
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

      return [flattenSettingsResponse(environmentId, result.buildSettings, result.runtimeSettings)];
    },
    getKey: (item) => item.environmentId,
    id: "environmentSettings",
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];
      await dispatchSettingsMutations(original, modified);
    },
  }),
);

export type EnvironmentSettings = z.infer<typeof schema>;

type SettingsResponse = Awaited<ReturnType<typeof trpcClient.deploy.environmentSettings.get.query>>;

function changed<T>(a: T, b: T): boolean {
  return JSON.stringify(a) !== JSON.stringify(b);
}

function extractKeyspaceIds(config: SentinelConfig | undefined): string[] {
  return config?.policies.flatMap((p) => p.keyauth.keySpaceIds) ?? [];
}

function flattenSettingsResponse(
  environmentId: string,
  build: SettingsResponse["buildSettings"],
  runtime: SettingsResponse["runtimeSettings"],
): EnvironmentSettings {
  return {
    environmentId,
    dockerfile: build?.dockerfile ?? "Dockerfile",
    dockerContext: build?.dockerContext ?? ".",
    port: runtime?.port ?? 8080,
    cpuMillicores: runtime?.cpuMillicores ?? 256,
    memoryMib: runtime?.memoryMib ?? 256,
    command: runtime?.command ?? [],
    healthcheck: runtime?.healthcheck ?? null,
    regionConfig: runtime?.regionConfig ?? {},
    shutdownSignal: "SIGTERM",
    sentinelConfig: runtime?.sentinelConfig,
  };
}

async function dispatchSettingsMutations(
  original: EnvironmentSettings,
  modified: EnvironmentSettings,
): Promise<void> {
  const { environmentId } = original;
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

  if (changed(original.command, modified.command)) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateCommand.mutate({
        environmentId,
        command: modified.command,
      }),
    );
  }

  if (changed(original.healthcheck, modified.healthcheck)) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateHealthcheck.mutate({
        environmentId,
        healthcheck: modified.healthcheck,
      }),
    );
  }

  const origRegions = Object.keys(original.regionConfig).sort();
  const modRegions = Object.keys(modified.regionConfig).sort();
  const regionsChanged = changed(origRegions, modRegions);

  if (regionsChanged) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateRegions.mutate({
        environmentId,
        regions: modRegions,
      }),
    );
  }

  const origValues = Object.values(original.regionConfig);
  const modValues = Object.values(modified.regionConfig);
  const instancesChanged =
    !regionsChanged &&
    origValues.length === modValues.length &&
    modValues.length > 0 &&
    modValues[0] !== origValues[0];

  if (instancesChanged) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateInstances.mutate({
        environmentId,
        replicasPerRegion: modValues[0],
      }),
    );
  }

  if (changed(original.sentinelConfig, modified.sentinelConfig)) {
    mutations.push(
      trpcClient.deploy.environmentSettings.sentinel.updateMiddleware.mutate({
        environmentId,
        keyspaceIds: extractKeyspaceIds(modified.sentinelConfig),
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
}
