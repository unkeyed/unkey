"use client";
import type { SentinelConfig } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { useSyncExternalStore } from "react";
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
  watchPaths: z.array(z.string()).default([]),
  // Runtime settings
  port: z.number().int(),
  cpuMillicores: z.number().int(),
  memoryMib: z.number().int(),
  command: z.array(z.string()),
  healthcheck: healthcheckSchema,
  regions: z.array(z.object({ id: z.string(), name: z.string(), replicas: z.number().int() })),
  shutdownSignal: z.string(),
  sentinelConfig: sentinelConfigSchema,
  openapiSpecPath: z.string().nullable().default(null),
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

      return [
        flattenSettingsResponse(
          environmentId,
          result.buildSettings,
          result.runtimeSettings,
          result.regionalSettings,
        ),
      ];
    },
    getKey: (item) => item.environmentId,
    id: "environmentSettings",
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];
      const silent = transaction.metadata?.silent === true;
      await dispatchSettingsMutations(original, modified, silent);
    },
  }),
);

export type EnvironmentSettings = z.infer<typeof schema>;

/** Default values for environment settings fields (excluding regions, which are runtime-dependent). */
export const ENVIRONMENT_SETTINGS_DEFAULTS = {
  dockerfile: "Dockerfile",
  dockerContext: ".",
  port: 8080,
  cpuMillicores: 250,
  memoryMib: 256,
  shutdownSignal: "SIGTERM",
} as const;

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
  regional: SettingsResponse["regionalSettings"],
): EnvironmentSettings {
  const d = ENVIRONMENT_SETTINGS_DEFAULTS;
  return {
    environmentId,
    dockerfile: build?.dockerfile || d.dockerfile,
    dockerContext: build?.dockerContext || d.dockerContext,
    watchPaths: build?.watchPaths ?? [],
    port: runtime?.port ?? d.port,
    cpuMillicores: runtime?.cpuMillicores ?? d.cpuMillicores,
    memoryMib: runtime?.memoryMib ?? d.memoryMib,
    command: runtime?.command ?? [],
    healthcheck: runtime?.healthcheck ?? null,
    regions: regional
      .filter((r): r is typeof r & { region: NonNullable<typeof r.region> } => r.region !== null)
      .map((r) => ({
        id: r.region.id,
        name: r.region.name,
        replicas: r.replicas,
      })),
    shutdownSignal: d.shutdownSignal,
    sentinelConfig: runtime?.sentinelConfig,
    openapiSpecPath: runtime?.openapiSpecPath ?? null,
  };
}

/**
 * Build an array of tRPC mutation promises for settings that changed between
 * `original` and `modified`, targeting `environmentId`.
 *
 * Pure function — no toasts, no side-effects beyond the network calls.
 */
export function buildSettingsMutations(
  environmentId: string,
  original: EnvironmentSettings,
  modified: EnvironmentSettings,
): Promise<unknown>[] {
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

  if (changed(original.watchPaths, modified.watchPaths)) {
    mutations.push(
      trpcClient.deploy.environmentSettings.build.updateWatchPaths.mutate({
        environmentId,
        watchPaths: modified.watchPaths,
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

  const origRegionIds = original.regions.map((r) => r.id).sort();
  const modRegionIds = modified.regions.map((r) => r.id).sort();
  const regionsChanged = changed(origRegionIds, modRegionIds);

  if (regionsChanged) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateRegions.mutate({
        environmentId,
        regionIds: modRegionIds,
      }),
    );
  }

  const origReplicas = original.regions.at(0)?.replicas ?? 1;
  const modReplicas = modified.regions.at(0)?.replicas ?? 1;
  const instancesChanged =
    !regionsChanged && modified.regions.length > 0 && modReplicas !== origReplicas;

  if (instancesChanged) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateInstances.mutate({
        environmentId,
        replicasPerRegion: modReplicas,
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

  if (modified.openapiSpecPath !== original.openapiSpecPath) {
    mutations.push(
      trpcClient.deploy.environmentSettings.runtime.updateOpenapiSpecPath.mutate({
        environmentId,
        openapiSpecPath: modified.openapiSpecPath,
      }),
    );
  }

  return mutations;
}

async function dispatchSettingsMutations(
  original: EnvironmentSettings,
  modified: EnvironmentSettings,
  silent = false,
): Promise<void> {
  const mutations = buildSettingsMutations(original.environmentId, original, modified);

  if (mutations.length === 0) {
    return;
  }

  const allMutations = Promise.all(mutations);
  if (!silent) {
    toast.promise(allMutations, {
      loading: "Saving settings...",
      success: "Settings updated",
      error: (err) => ({
        message: "Failed to update settings",
        description: err instanceof Error ? err.message : "An unexpected error occurred",
      }),
    });
  }
  saveStore.pendingSaves++;
  saveStore.notify();
  try {
    await allMutations;
    saveStore.savedCount++;
    saveStore.notify();
  } finally {
    saveStore.pendingSaves--;
    saveStore.notify();
  }
}

/**
 * Store for tracking in-flight and completed settings saves.
 *
 * Grouped into a single object so the boundary is obvious and
 * `dispatchSettingsMutations` has one place to update.
 * Consumers subscribe via `useSyncExternalStore` — no React context needed
 * because settings mutations always originate from this module.
 */
const saveStore = {
  pendingSaves: 0,
  savedCount: 0,
  listeners: new Set<() => void>(),
  notify() {
    for (const cb of this.listeners) {
      cb();
    }
  },
  subscribe(cb: () => void): () => void {
    this.listeners.add(cb);
    return () => {
      this.listeners.delete(cb);
    };
  },
};

export function useSettingsIsSaving(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.pendingSaves > 0,
  );
}

/** Returns true once at least one settings save has completed in this session. */
export function useSettingsHasSaved(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.savedCount > 0,
  );
}
